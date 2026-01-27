import json
import httpx
import asyncio
from typing import AsyncGenerator, Callable, Optional

# # ============ Streaming Chat Functions ============

# async def get_chatbot_response_stream(
#     client, 
#     model_name: str, 
#     messages: list,
#     temperature: float = 0,
#     on_chunk: Optional[Callable[[str], None]] = None
# ) -> AsyncGenerator[str, None]:

#     response = await client.chat.completions.create(
#         model=model_name,
#         messages=messages,
#         temperature=temperature,
#         top_p=0.1,
#         max_tokens=8192,
#         stream=True,  # Enable streaming
#     )
    
#     async for chunk in response:
#         if chunk.choices[0].delta.content:
#             text = chunk.choices[0].delta.content
#             if on_chunk:
#                 on_chunk(text)
#             yield text


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
        stream_options={"include_usage": True}  # Get usage at the end
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
        
        # Usage info (comes at the end with stream_options)
        if hasattr(chunk, 'usage') and chunk.usage:
            yield {
                "type": "usage",
                "input_tokens": chunk.usage.prompt_tokens,
                "output_tokens": chunk.usage.completion_tokens,
                "total_tokens": chunk.usage.total_tokens,
                "full_content": full_content
            }
            print("input_tokens : " ,chunk.usage.prompt_tokens)
            print("output_tokens : " ,chunk.usage.completion_tokens)
            print("total_tokens : " ,chunk.usage.total_tokens)


# ============ Non-Streaming Functions (Original) ============

async def get_chatbot_response(client, model_name, messages, temperature=0):
    """Original non-streaming response"""
    response = await client.chat.completions.create(
        model=model_name,
        messages=messages,
        temperature=temperature,
        top_p=0.1,
        max_tokens=8192,
        response_format={"type": "json_object"}
    )
    return response.choices[0].message.content


async def get_chatbot_full_response(client, model_name, messages, temperature=0):
    """Original non-streaming full response"""
    response = await client.chat.completions.create(
        model=model_name,
        messages=messages,
        temperature=temperature,
        top_p=0.1,
        max_tokens=8192,
        response_format={"type": "json_object"}
    )
    return response


# ============ Embedding Functions ============

async def get_embedding(RUNPOD_API_KEY, RUNPOD_EMBEDDING_URL, texts):
    headers = {
        'Content-Type': 'application/json',
        'Authorization': f"Bearer {RUNPOD_API_KEY}"
    }

    data = {"input": {"input": texts}}
    async with httpx.AsyncClient(timeout=60.0) as http_client:
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


# ============ Guard Functions ============

def evaluate_safety_response(guard_output: str, SAFETY_TAXONOMY: dict):
    if "unsafe" in guard_output:
        parts = guard_output.split('\n')
        if len(parts) > 1:
            code = parts[1].strip()
            reason = SAFETY_TAXONOMY.get(code, "Unknown Reason")
            return False, json.dumps({"decision": "not allowed", "message":f"Layer 1 : Unsafe ({code}) - {reason}"})
        else:
            return False, json.dumps({"decision": "not allowed", "message":f"Layer 1 : Unsafe - No specific code provided"})
    return True, "Guard Layer 1 : Safe....."


def check_prompt_attack(score: str, threshold: float = 0.9):
    try:
        score = float(score)
        if score > threshold:
            return False, json.dumps({"decision": "not allowed", "message":"Layer 2 : Jailbreak/Injection attack detected!"})
        else:
            return True, "Guard Layer 2 : Safe....."
    except ValueError:
        return False, json.dumps({"decision": "not allowed", "message":"Layer 2 : Error: Could not parse score."})


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
        print(f"Prompt length : {len(long_text)} -> Not Chunking... ")
        score_output = await get_guard_classification(
            client=client,
            model_name=model_name,
            text=long_text
        )
        return check_prompt_attack(score=score_output)
    else:
        # Condition 2 : if prompt > chunk size
        print(f"Prompt length: {len(long_text)} -> Chunking...")
        step = CHUNK_CHARS - OVERLAP_CHARS

        for i in range(0, len(long_text), step):
            chunk_text = long_text[i: i + CHUNK_CHARS]
            score_output = await get_guard_classification(
                client=client,
                model_name=model_name,
                text=chunk_text
            )
            is_safe, result_msg = check_prompt_attack(score=score_output)

            if not is_safe:
                print(f"   -> Attack found in chunk {i}-{i+len(chunk_text)}! , score = {score_output}")
                return is_safe, result_msg

        return True, "Guard Layer 2 : All chunks are Safe."