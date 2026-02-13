from dotenv import load_dotenv
import os, json
from copy import deepcopy
from .utils  import get_chatbot_response, get_embedding, get_closest_result, get_chatbot_full_response
from .prompts import PLANNER_SYSTEM_PROMPT, STEP_DEFINER_SYSTEM_PROMPT, DETAIL_SYSTEM_PROMTPS
from openai import AsyncOpenAI
from pinecone import Pinecone
from json_repair import repair_json
load_dotenv()


def dedup_relevant_text_by_section(text: str) -> str:

    lines = text.splitlines()
    blocks = []
    current_title = None
    current_lines = []

    def flush_block():
        nonlocal current_title, current_lines
        if current_title is not None and current_lines:
            blocks.append((current_title, "\n".join(current_lines).strip()))
        current_title = None
        current_lines = []

    for line in lines:
        stripped = line.strip()
        # ดูหัวบรรทัดที่ขึ้นต้นว่า "มาตรา " (ทั้งเลขไทยเลขอารบิกใช้ได้หมด เพราะเราใช้ทั้งบรรทัดเป็น key)
        if stripped.startswith("มาตรา "):
            # ปิด block เดิมก่อน (ถ้ามี)
            flush_block()
            # เปิด block ใหม่
            current_title = stripped  # ใช้ทั้งบรรทัดเป็นชื่อมาตรา เช่น "มาตรา 1184" หรือ "มาตรา ๑๑๒๙ วรรคสอง"
            current_lines = [stripped]
        else:
            # อยู่ใน block เดิมก็เก็บต่อ
            if current_title is not None:
                current_lines.append(line)

    # flush block สุดท้าย
    flush_block()

    # ตอนนี้ blocks = [(title, block_text), ...]
    # ทำ dedup ตาม title (มาตรา)
    seen = set()
    deduped_blocks = []
    for title, block_text in blocks:
        if title not in seen:
            seen.add(title)
            deduped_blocks.append(block_text)

    # join กลับเป็นข้อความยาว ๆ (คั่นด้วย \n\n)
    return "\n\n".join(deduped_blocks)

class DetailsAgent():
    def __init__(self):
        self.client = AsyncOpenAI(
            api_key = os.getenv("TYPHOON_API"),
            base_url = os.getenv("RUNPOD_CHATBOT_URL")
        )
        self.token = os.getenv("RUNPOD_API_KEY")
        self.embedding_url = os.getenv("RUNPOD_EMBEDDING_URL")
        self.model_name = os.getenv("MODEL_NAME")
        self.pc = Pinecone(api_key = os.getenv("PINECONE_API_KEY"))
        self.index_name = os.getenv("PINECONE_INDEX_NAME")
        
        self.PLANNER_SYSTEM_PROMPT = PLANNER_SYSTEM_PROMPT
        self.STEP_DEFINER_SYSTEM_PROMPT = STEP_DEFINER_SYSTEM_PROMPT
        self.system_prompt = DETAIL_SYSTEM_PROMTPS
        self.totalInputTokens = 0
        self.totalOutputTokens = 0
        self.finalOutputTokens = 0
        self.totalUsedTokens = 0

    def postprocess(self, 
                    output,
                    totalInputTokens,
                    totalOutputTokens,
                    finalOutputTokens,
                    totalUsedTokens):
        outputJSON = repair_json(output)
        output = json.loads(outputJSON)
        dict_output = {
            "role" : "assistant",
            "content" : output['ans'],
            "memory" : {
                "agent" : "detail agent",
                "sections" : output['sections']
            },
            "totalInputTokens" :totalInputTokens,
            "totalOutputTokens":totalOutputTokens,
            "finalOutputTokens" :finalOutputTokens,
            "totalUsedTokens" :totalUsedTokens
        }
        return dict_output

    def get_detail(self, list_closest:  list) -> str:
        texts = []
        for m in list_closest:
            t = (m.get("metadata", {}) or {}).get("text", "")
            t = t.strip()
            if t:
                texts.append(t)

        combined = "\n\n".join(texts)
        return combined
    def joined_text(self, history_text):
        self.joined_history_text = "".join(
            f"question: {item['question']}\n"
            f"sections: {item['sections']}\n"
            f"ans: {item['ans']}\n\n"
            for item in history_text
        )

    def join_step(self, history_text):
        for index, item in enumerate(history_text):
            self.joined_step += (
                f"{index+1}.{item['question']}\n"
            )
            
    async def COT(self, question: str, response):
        for index, step in enumerate(response["steps"]):
            # print(f"COT definer : {index+1}")
            # Step : Definer
            definer_prompt = (
                f"Original question:\n{question}\n\n"
                f"Current plan step:\n{step}\n\n"
                "Completed steps summary (may be empty):\n"
                f"{self.joined_history_text}\n\n"
                "Your job:\n"
                "- Choose the next retrieval task for this step.\n"
                "- Use 'search' when you need to retrieve new legal text.\n"
                "- Use 'aggregate' when you need to combine or compare previous findings.\n"
                "- Use 'verify' when you only need to confirm or check something against existing information.\n\n"
                "Return ONLY a single JSON object. No explanation, no markdown, no extra text.\n"
                "Use this EXACT schema:\n"
                "{\n"
                '  "task_type": "search",\n'
                '  "query": "detailed retrieval or aggregation query in Thai",\n'
                '  "notes": "brief reasoning (<=3 sentences)"\n'
                "}\n"
            )
            messages_definer = [
                {"role":"system","content": self.STEP_DEFINER_SYSTEM_PROMPT},
                {"role":"user","content": definer_prompt},
            ]
            response_definer, inputTokens, outputTokens, totalTokens = await get_chatbot_response(self.client, self.model_name, messages_definer)
            self.totalInputTokens += inputTokens
            self.totalOutputTokens += outputTokens
            self.totalUsedTokens += totalTokens
            response_definer = repair_json(response_definer)
            response_definer = json.loads(response_definer)
            # print(response_definer)
            
            # Step : Retrive, QA
            embedding_result = await get_embedding(self.token, self.embedding_url, response_definer['query'])
            vector = embedding_result[0]
            result = await get_closest_result(self.pc, self.index_name, vector)
            
            matches = result.get("matches", [])
            docs_str = self.get_detail(matches)
            self.list_docs_str.append(docs_str)

            QA_prompt = (
                f"Original question: {question}\n"
                f"Subquery: {response_definer['query']}\n"
                f"Retrieved context:\n{docs_str}\n\n"
                "ตอบคำถามตามรูปแบบ JSON ที่กำหนด (sections, ans) โดยอาศัยบริบทและมาตรากฎหมายที่พบ."
            )
            messages_QA = [
                {"role":"system","content": self.system_prompt},
                {"role":"user","content": QA_prompt},
            ]
            response_qa, inputTokens, outputTokens, totalTokens = await get_chatbot_response(self.client, self.model_name, messages_QA)
            self.totalInputTokens += inputTokens
            self.totalOutputTokens += outputTokens
            self.totalUsedTokens += totalTokens
            response_qa = repair_json(response_qa)
            response_qa = json.loads(response_qa)
            self.history_text.append(response_qa)
            self.joined_text(self.history_text)
    
            # print(f"question : {response_qa['question']}")
            # print(f"sections : {response_qa['sections']}")
            # print(f"ANS : {response_qa['ans']}")
        self.join_step(self.history_text)
    
    
    async def get_response_COT(self, question: str):
        self.list_docs_str = []
        self.history_text = []
        self.joined_history_text = ""
        self.joined_step = ""
        
        question = deepcopy(question)
        # print("Plan Agent : Start ....")
        user_prompt = (
            "You are a planning specialist for a legal Retrieval-Augmented Generation pipeline.\n"
            "Deconstruct the user's question into a minimal sequence of retrieval steps.\n\n"
            f"Question:\n{question}\n\n"
            "Instructions:\n"
            "- Identify whether the query is single-hop or requires multiple steps.\n"
            "- Produce Limit 3 ordered steps that resolve the question via retrieval.\n"
            "- Each step must be a concrete sub-question or aggregation task (no verification-only steps).\n\n"
            "Return ONLY a single JSON object. No explanations, no markdown, no extra text.\n"
            "Use this EXACT schema:\n"
            "{\n"
            '  "rationale": "string",\n'
            '  "steps": ["step 1", "step 2", "..."]\n'
            "}\n"
        )
        messages = [
            {"role":"system","content": self.PLANNER_SYSTEM_PROMPT},
            {"role":"user","content": user_prompt},
        ]
        response, inputTokens, outputTokens, totalTokens = await get_chatbot_response(self.client, self.model_name, messages)
        self.totalInputTokens += inputTokens
        self.totalOutputTokens += outputTokens
        self.totalUsedTokens += totalTokens
        response = repair_json(response)
        response_plan = json.loads(response)
        # print("Plan Agent : ")
        # for index, step in enumerate(response_plan["steps"]):
        #     print( f"{index+1}: {step}")
        # print("Plan Agent : End ....")
        
        # print("COT Agent : Start COT ....")
        await self.COT(
            question = question,
            response = response_plan
        )
        # print("COT Agent : End COT ....")

        # print("Start combine text.......")
        joined_text = "\n\n".join(self.list_docs_str)
        text = dedup_relevant_text_by_section(joined_text)

        # print("Start create detail content .......")
        detail_content = f"""
            Original question:
            {question}

            Plan:
            {self.joined_step}
            
            Step outcomes:
            {self.joined_history_text}

            Rules:
            1. "sections":
                - List all relevant legal provisions (มาตรา) by number only.
                - Example: "มาตรา ๒๔๓, มาตรา ๒๔๔".

            2. "ans":
                - First, write out the full text of each relevant section, using ONLY the wording that appears in the Relevant text above.
                    Example format:
                    มาตรา ๒๔๓ [เนื้อหาของมาตราตามที่ปรากฏใน Relevant text]
                - After listing all sections, write a clear Thai explanation that answers the Original question,
                    based strictly on those sections. The explanation must be in Thai.
                - Do NOT invent or modify any legal text. Use only sentences that already appear in the Relevant text.

            3. If the information in Relevant text is insufficient or you cannot determine the answer:
                - Set "sections" to "ไม่ทราบ"
                - Set "ans" to "ไม่ทราบ"
        """

        messages = [
            {"role":"system","content": self.system_prompt},
            {"role":"user","content": detail_content},
        ]
        # print("Detail Agent : user content")
        # print("Detail Agent : Prompting......")
        chatbot_output = await get_chatbot_full_response(self.client, self.model_name, messages)
        self.totalInputTokens += chatbot_output.usage.prompt_tokens
        self.totalOutputTokens += chatbot_output.usage.completion_tokens
        self.finalOutputTokens = chatbot_output.usage.completion_tokens
        self.totalUsedTokens += chatbot_output.usage.total_tokens
        output = self.postprocess(output       = chatbot_output.choices[0].message.content,
                                  totalInputTokens  = self.totalInputTokens,
                                  totalOutputTokens = self.totalOutputTokens,
                                  finalOutputTokens = self.finalOutputTokens,
                                  totalUsedTokens  = self.totalUsedTokens)
        self.totalInputTokens = 0
        self.totalOutputTokens = 0
        self.finalOutputTokens = 0
        self.totalUsedTokens = 0
        return output
    
    async def get_response_non_COT(self, question: str):
        question = deepcopy(question)
        # print("Detail Agent : Start non-COT ....")
        # print("Detail Agent : Embedding ......")
        embedding_result = await get_embedding(self.token, self.embedding_url, [question])
        vector = embedding_result[0]
        
        # print("Detail Agent : Get closest ......")
        result = await get_closest_result(self.pc, self.index_name, vector)
        
        # print("Detail Agent : Get Content from Vector database")
        matches = result.get("matches", [])
        docs_str = self.get_detail(matches)
        
        detail_content = f"""
            Question:
            {question}
            
            Relevant text:
            {docs_str}
            
            Rules:
            1. "sections":
                - List all relevant legal provisions (มาตรา) by number only.
                - Example: "มาตรา ๒๔๓, มาตรา ๒๔๔".

            2. "ans":
                - First, write out the full text of each relevant section, using ONLY the wording that appears in the Relevant text above.
                    Example format:
                    มาตรา ๒๔๓ [เนื้อหาของมาตราตามที่ปรากฏใน Relevant text]
                - After listing all sections, write a clear Thai explanation that answers the Original question,
                    based strictly on those sections. The explanation must be in Thai.
                - Do NOT invent or modify any legal text. Use only sentences that already appear in the Relevant text.

            3. If the information in Relevant text is insufficient or you cannot determine the answer:
                - Set "sections" to "ไม่ทราบ"
                - Set "ans" to "ไม่ทราบ"
        """
        messages = [
            {"role":"system","content": self.system_prompt},
            {"role":"user","content": detail_content},
        ]
        # print("Detail Agent : non-COT Prompting ......")
        chatbot_output = await get_chatbot_full_response(self.client, self.model_name, messages)
        output = self.postprocess(output       = chatbot_output.choices[0].message.content,
                                  totalInputTokens  = chatbot_output.usage.prompt_tokens,
                                  totalOutputTokens = chatbot_output.usage.completion_tokens,
                                  finalOutputTokens = chatbot_output.usage.completion_tokens,
                                  totalUsedTokens  = chatbot_output.usage.total_tokens)
        return output