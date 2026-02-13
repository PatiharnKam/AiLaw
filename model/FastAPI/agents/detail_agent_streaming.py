from dotenv import load_dotenv
import os
import json
from copy import deepcopy
from typing import AsyncGenerator
from .utils import (
    get_chatbot_response,
    get_chatbot_full_response_stream,
    get_embedding,
    get_closest_result
)
from .prompts import PLANNER_SYSTEM_PROMPT, STEP_DEFINER_SYSTEM_PROMPT, DETAIL_STREAMING_SYSTEM_PROMTPS
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
        if stripped.startswith("มาตรา "):
            flush_block()
            current_title = stripped
            current_lines = [stripped]
        else:
            if current_title is not None:
                current_lines.append(line)

    flush_block()

    seen = set()
    deduped_blocks = []
    for title, block_text in blocks:
        if title not in seen:
            seen.add(title)
            deduped_blocks.append(block_text)

    return "\n\n".join(deduped_blocks)


class DetailsAgentStreaming:
    def __init__(self):
        self.client = AsyncOpenAI(
            api_key=os.getenv("TYPHOON_API"),
            base_url=os.getenv("RUNPOD_CHATBOT_URL")
        )
        self.token = os.getenv("RUNPOD_API_KEY")
        self.embedding_url = os.getenv("RUNPOD_EMBEDDING_URL")
        self.model_name = os.getenv("MODEL_NAME")
        self.pc = Pinecone(api_key=os.getenv("PINECONE_API_KEY"))
        self.index_name = os.getenv("PINECONE_INDEX_NAME")

        self.PLANNER_SYSTEM_PROMPT = PLANNER_SYSTEM_PROMPT
        self.STEP_DEFINER_SYSTEM_PROMPT = STEP_DEFINER_SYSTEM_PROMPT
        self.streaming_system_prompt = DETAIL_STREAMING_SYSTEM_PROMTPS
        self.totalInputTokens = 0
        self.totalOutputTokens = 0
        self.finalOutputTokens = 0
        self.totalUsedTokens = 0


    def get_detail(self, list_closest: list) -> str:
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
            # f"question: {item['question']}\n"
            f"sections: {item['sections']}\n"
            f"ans: {item['ans']}\n\n"
            for item in history_text
        )

    def join_step(self, history_text):
        for index, item in enumerate(history_text):
            # self.joined_step += f"{index+1}.{item['question']}\n"
            self.joined_step += f"{index+1}.{item}\n"

    # ============ Non-COT Streaming ============
    
    async def get_response_non_COT_stream(
        self, 
        question: str
    ) -> AsyncGenerator[dict, None]:
        question = deepcopy(question)
        # print("Detail Agent (Streaming) : Start non-COT ....")
        # print("Detail Agent (Streaming) : Embedding ......")
        yield {"type": "status", "message": "กำลังค้นหาข้อมูล..."}
        
        embedding_result = await get_embedding(self.token, self.embedding_url, [question])
        vector = embedding_result[0]

        # print("Detail Agent : Get closest ......")
        result = await get_closest_result(self.pc, self.index_name, vector)

        # print("Detail Agent : Get Content from Vector database")
        matches = result.get("matches", [])
        docs_str = self.get_detail(matches)

        yield {"type": "status", "message": "กำลังวิเคราะห์และสร้างคำตอบ..."}

        # Step 4: Build prompt
        detail_content = f"""
            Question:
            {question}
            
            Relevant text:
            {docs_str}
            
            Rules:
            1. "มาตราที่เกี่ยวข้อง":
                - List all relevant legal provisions (มาตรา) by number only.
                - Example: "มาตรา ๒๔๓, มาตรา ๒๔๔".

            2. "คำตอบ":
                - First, write out the full text of each relevant section, using ONLY the wording that appears in the Relevant text above.
                    Example format:
                    มาตรา ๒๔๓ [เนื้อหาของมาตราตามที่ปรากฏใน Relevant text]
                - After listing all sections, write a clear Thai explanation that answers the Original question,
                    based strictly on those sections. The explanation must be in Thai.
                - Do NOT invent or modify any legal text. Use only sentences that already appear in the Relevant text.

            3. If the information in Relevant text is insufficient or you cannot determine the answer:
                - Set "มาตราที่เกี่ยวข้อง" to "ไม่ทราบ"
                - Set "คำตอบ" to "ไม่ทราบ"
        """

        messages = [
            {"role": "system", "content": self.streaming_system_prompt},
            {"role": "user", "content": detail_content},
        ]

        # print("Detail Agent : Streaming response ......")
        full_content = ""
        input_tokens = 0
        output_tokens = 0

        async for chunk in get_chatbot_full_response_stream(
            self.client, 
            self.model_name, 
            messages
        ):
            if chunk["type"] == "content":
                full_content += chunk["text"]
                yield {
                    "type": "content",
                    "text": chunk["text"]
                }
            elif chunk["type"] == "usage":
                input_tokens = chunk["inputTokens"]
                output_tokens = chunk["outputTokens"]
                total_tokens = chunk["totalTokens"]

        self.totalInputTokens += input_tokens
        self.totalOutputTokens += output_tokens
        self.finalOutputTokens = output_tokens
        self.totalUsedTokens += total_tokens
        # Step 6: Yield completion
        yield {
            "type": "done",
            "totalInputTokens": self.totalInputTokens,
            "totalOutputTokens": self.totalOutputTokens,
            "finalOutputTokens": self.finalOutputTokens,
            "totalUsedTokens": self.totalUsedTokens,
            "fullContent": full_content
        }

        # print("Detail Agent (Streaming) : End non-COT ....")

    # ============ COT Streaming ============

    async def _run_COT_steps(
        self, 
        question: str, 
        response_plan: dict
    ) -> AsyncGenerator[dict, None]:
        """Run COT steps and yield progress updates"""
        self.list_docs_str = []
        self.history_text = []
        self.joined_history_text = ""
        self.joined_step = ""

        total_steps = len(response_plan["steps"])

        for index, step in enumerate(response_plan["steps"]):
            # print(f"COT definer : {index+1}")
            
            yield {
                "type": "cot_step",
                "step": index + 1,
                "total": total_steps,
                "description": step
            }

            yield {"type": "status", "message": f"กำลังปรับแต่งรอบที่ {index+1}"}

            # Step Definer
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
                "{\n"
                '  "task_type": "search",\n'
                '  "query": "detailed retrieval or aggregation query in Thai",\n'
                '  "notes": "brief reasoning (<=3 sentences)"\n'
                "}\n"
            )

            messages_definer = [
                {"role": "system", "content": self.STEP_DEFINER_SYSTEM_PROMPT},
                {"role": "user", "content": definer_prompt},
            ]

            response_definer, inputTokens, outputTokens, totalTokens = await get_chatbot_response(self.client, self.model_name, messages_definer)
            self.totalInputTokens += inputTokens
            self.totalOutputTokens += outputTokens
            self.totalUsedTokens += totalTokens
            response_definer = repair_json(response_definer)
            response_definer = json.loads(response_definer)

            yield {"type": "status", "message": f"กำลังค้นหาข้อมูลรอบที่ {index+1}"}
            # Retrieve documents
            embedding_result = await get_embedding(self.token, self.embedding_url, response_definer['query'])
            vector = embedding_result[0]
            result = await get_closest_result(self.pc, self.index_name, vector)

            matches = result.get("matches", [])
            docs_str = self.get_detail(matches)
            self.list_docs_str.append(docs_str)

            # print(f"COT QA : {index+1}")
            yield {"type": "status", "message": f"กำลังวิเคราะห์และสร้างคำตอบรอบที่ {index+1}"}
            QA_prompt = (
                f"Original question: {question}\n"
                f"Subquery: {response_definer['query']}\n"
                f"Retrieved context:\n{docs_str}\n\n"
                "ตอบคำถามตามรูปแบบ JSON ที่กำหนด (sections, ans) โดยอาศัยบริบทและมาตรากฎหมายที่พบ."
            )

            messages_QA = [
                {"role": "system", "content": self.streaming_system_prompt},
                {"role": "user", "content": QA_prompt},
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
            # print(f"Step {index+1} completed: {response_qa.get('question', 'N/A')}")
        self.join_step(self.history_text)

    async def get_response_COT_stream(
        self, 
        question: str
    ) -> AsyncGenerator[dict, None]:
        self.list_docs_str = []
        self.history_text = []
        self.joined_history_text = ""
        self.joined_step = ""
        
        question = deepcopy(question)
        # print("Plan Agent : Start ....")
        yield {"type": "status", "message": "กำลังวางแผนการค้นหา..."}

        # Step 1: Planning
        user_prompt = (
            "You are a planning specialist for a legal Retrieval-Augmented Generation pipeline.\n"
            "Deconstruct the user's question into a minimal sequence of retrieval steps.\n\n"
            f"Question:\n{question}\n\n"
            "Instructions:\n"
            "- Identify whether the query is single-hop or requires multiple steps.\n"
            "- Produce Limit 3 ordered steps that resolve the question via retrieval.\n"
            "- Each step must be a concrete sub-question or aggregation task (no verification-only steps).\n\n"
            "Return ONLY a single JSON object. No explanations, no markdown, no extra text.\n"
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

        yield {
            "type": "plan",
            "steps": response_plan["steps"],
            "rationale": response_plan.get("rationale", "")
        }
        # print("Plan Agent : ")
        # for index, step in enumerate(response_plan["steps"]):
        #     print( f"{index+1}: {step}")
        # print("Plan Agent : End ....")

        # Step 2: Execute COT steps
        async for update in self._run_COT_steps(question, response_plan):
            yield update

        yield {"type": "status", "message": "กำลังสรุปคำตอบ..."}

        # print("Start combine text.......")
        # Step 3: Combine and generate final answer
        joined_text = "\n\n".join(self.list_docs_str)
        text = dedup_relevant_text_by_section(joined_text)

        
        detail_content = f"""
            Original question:
            {question}

            Plan:
            {self.joined_step}
            
            Step outcomes:
            {self.joined_history_text}

            Rules:
            1. "มาตราที่เกี่ยวข้อง":
                - List all relevant legal provisions (มาตรา) by number only.
                - Example: "มาตรา ๒๔๓, มาตรา ๒๔๔".

            2. "คำตอบ":
                - First, write out the full text of each relevant section, using ONLY the wording that appears in the Relevant text above.
                    Example format:
                    มาตรา ๒๔๓ [เนื้อหาของมาตราตามที่ปรากฏใน Relevant text]
                - After listing all sections, write a clear Thai explanation that answers the Original question,
                    based strictly on those sections. The explanation must be in Thai.
                - Do NOT invent or modify any legal text. Use only sentences that already appear in the Relevant text.

            3. If the information in Relevant text is insufficient or you cannot determine the answer:
                - Set "มาตราที่เกี่ยวข้อง" to "ไม่ทราบ"
                - Set "คำตอบ" to "ไม่ทราบ"
        """

        messages = [
            {"role": "system", "content": self.streaming_system_prompt},
            {"role": "user", "content": detail_content},
        ]

        # Step 4: Stream final response
        # print("Detail Agent : Streaming final response ......")
        full_content = ""
        input_tokens = 0
        output_tokens = 0

        async for chunk in get_chatbot_full_response_stream(
            self.client, 
            self.model_name, 
            messages
        ):
            if chunk["type"] == "content":
                full_content += chunk["text"]
                yield {
                    "type": "content",
                    "text": chunk["text"]
                }
            elif chunk["type"] == "usage":
                input_tokens = chunk["inputTokens"]
                output_tokens = chunk["outputTokens"]
                total_tokens = chunk["totalTokens"]

        self.totalInputTokens += input_tokens
        self.totalOutputTokens += output_tokens
        self.finalOutputTokens = output_tokens
        self.totalUsedTokens += total_tokens
        # Step 5: Yield completion
        yield {
            "type": "done",
            "totalInputTokens": self.totalInputTokens,
            "totalOutputTokens": self.totalOutputTokens,
            "finalOutputTokens": self.finalOutputTokens,
            "totalUsedTokens": self.totalUsedTokens,
            "fullContent": full_content
        }

        self.totalInputTokens = 0
        self.totalOutputTokens = 0
        self.finalOutputTokens = 0
        self.totalUsedTokens = 0

        # print("Detail Agent (Streaming COT) : End ....")
