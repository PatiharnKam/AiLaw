import json
import tiktoken
import httpx
import asyncio
from .prompts import PROMPT_ATTACK_MESSAGE
from typing import AsyncGenerator

def count_tokens(text: str, model: str = "gpt-4") -> int:
    try:
        encoding = tiktoken.encoding_for_model(model)
    except KeyError:
        encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def count_messages_tokens(messages: list, model: str = "gpt-4") -> int:
    total = 0
    for msg in messages:
        total += count_tokens(msg.get("content", ""), model)
        total += 4  
    total += 2
    return total

async def get_chatbot_full_response_stream(
    client, 
    model_name: str, 
    messages: list,
    temperature: float = 0
) -> AsyncGenerator[dict, None]:

    response = await client.chat.completions.create(
        model=model_name,
        messages=messages,
        temperature=temperature,
        top_p=0.1,
        max_tokens=8192,
        stream=True,
    )
    
    full_content = ""
    
    async for chunk in response:
        # Content chunk
        if chunk.choices and chunk.choices[0].delta.content:
            text = chunk.choices[0].delta.content
            full_content += text
            yield {
                "type": "content",
                "text": text
            }
        
    inputTokens = count_messages_tokens(messages)
    outputTokens = count_tokens(full_content)
    totalTokens = inputTokens + outputTokens
    
    # print(f"final step Counted - input: {inputTokens}, output: {outputTokens}, total: {totalTokens}")
    
    yield {
        "type": "usage",
        "inputTokens": inputTokens,
        "outputTokens": outputTokens,
        "totalTokens": totalTokens,
        "fullContent": full_content
    }

async def get_chatbot_response(client, model_name, messages, temperature=0):
    response = await client.chat.completions.create(
        model=model_name,
        messages=messages,
        temperature=temperature,
        top_p=0.1,
        max_tokens=8192,
        response_format={"type": "json_object"}
    )

    inputTokens = count_messages_tokens(messages)
    outputTokens = count_tokens(response.choices[0].message.content)
    totalTokens = inputTokens + outputTokens
    # print(f"each step input: {inputTokens}, output: {outputTokens}, total: {totalTokens}")
    
    return response.choices[0].message.content, inputTokens, outputTokens, totalTokens

async def get_chatbot_full_response(client, model_name, messages, temperature=0):
    response = await client.chat.completions.create(
        model=model_name,
        messages=messages,
        temperature=temperature,
        top_p=0.1,
        max_tokens=8192,
        response_format={"type": "json_object"}
    )
    
    return response

async def get_embedding(RUNPOD_API_KEY, RUNPOD_EMBEDDING_URL, texts):
    headers = {
        'Content-Type': 'application/json',
        'Authorization': f"Bearer {RUNPOD_API_KEY}"
    }

    data = {"input": {"input": texts}}
    async with httpx.AsyncClient(timeout=None) as http_client:
        response = await http_client.post(
            RUNPOD_EMBEDDING_URL, 
            headers=headers, 
            json=data,
        )
        response.raise_for_status()
    resp = response.json()
    
    embeddings = [item["embedding"] for item in resp["output"]["data"]]
    return embeddings


async def get_closest_result(pc, index_name, input_embedding, top_k=10):
        loop = asyncio.get_event_loop()
    
        def _query():
            index = pc.Index(index_name)
            return index.query(
                vector=input_embedding,
                top_k=top_k,
                include_values=False,
                include_metadata=True
            )
    
        result = await loop.run_in_executor(None, _query)
        return result
    
def evaluate_safety_response(guard_output: str, SAFETY_TAXONOMY: dict, USER_FRIENDLY_MESSAGES: dict):
    if "unsafe" in guard_output:
        parts = guard_output.split('\n')
        if len(parts) > 1:
            code = parts[1].strip()
            reason = SAFETY_TAXONOMY.get(code, "Unknown Reason")
            user_message = USER_FRIENDLY_MESSAGES.get(code, USER_FRIENDLY_MESSAGES["default"])

            # print(f"[Guard Layer 1] BLOCKED - Code: {code}, Reason: {reason}")
            return False, json.dumps({"decision": "not allowed", "message":user_message})
        else:
            return False, json.dumps({"decision": "not allowed", "message":USER_FRIENDLY_MESSAGES["default"]})
    return True, "Guard Layer 1 : Safe....."

def check_prompt_attack(score: str, threshold: float = 0.9):
    try:
        score_float  = float(score)
        if score_float  > threshold:
            # print(f"[Guard Layer 2] BLOCKED - Jailbreak/Injection detected, score: {score_float}")
            return False, json.dumps({"decision": "not allowed", "message":PROMPT_ATTACK_MESSAGE})
        else:
            return True, "Guard Layer 2 : Safe....."
    except ValueError:
        return False, json.dumps({"decision": "not allowed", "message":"ขออภัย เกิดข้อผิดพลาดในการประมวลผล กรุณาลองใหม่อีกครั้ง"})
    
async def get_guard_classification(client, model_name: str, text: str):
    response = await client.chat.completions.create(
        model=model_name,
        messages=[{"role": "user", "content": text}]
    )
    return response.choices[0].message.content

async def chunked_safety_scan(client, model_name: str, long_text: str):
    CHUNK_CHARS = 800  
    OVERLAP_CHARS = 300

    # Condition 1 : if prompt < chunk size
    if len(long_text) <= CHUNK_CHARS:
        # print(f"Prompt length : {len(long_text)} -> Not Chunking... ")
        score_output = await get_guard_classification(
            client=client,
            model_name=model_name,
            text=long_text)
        return check_prompt_attack(score=score_output)
    else:
      # Condition 2 : if prompt > chunk size
    #   print(f"Prompt length: {len(long_text)}  -> Chunking...")
      step = CHUNK_CHARS - OVERLAP_CHARS
      
      for i in range(0, len(long_text), step):
        chunk_text = long_text[i : i + CHUNK_CHARS]
        # print(chunk_text)

        score_output = await get_guard_classification(
            client=client,
            model_name=model_name,
            text=chunk_text
        )
        # print(f"   - Checking chars {i}-{i+len(chunk_text)}: {score_output}")
        is_safe, result_msg = check_prompt_attack(score=score_output)

        if not is_safe:
        #   print(f"   -> Attack found in chunk {i}-{i+len(chunk_text)}! , score = {score_output}")
          return is_safe, result_msg

      return True, "Guard Layer 2 : All chunks are Safe."