import requests
import json
import httpx
import asyncio

async def get_chatbot_response(client, model_name, messages, temperature=0):
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
    async with httpx.AsyncClient(timeout=60.0) as http_client:
        response = await http_client.post(
            RUNPOD_EMBEDDING_URL, 
            headers=headers, 
            json=data,
        )
        response.raise_for_status()
    # response = requests.post(RUNPOD_EMBEDDING_URL, headers=headers, json=data)
    # response.raise_for_status()
    
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
        # index = pc.Index(index_name)
        
        # result = index.query(
        #     vector = input_embedding,
        #     top_k=top_k,
        #     include_values=False,
        #     include_metadata=True
        # )
        
        return result
    
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
            text=long_text)
        return check_prompt_attack(score=score_output)
    else:
      # Condition 2 : if prompt > chunk size
      print(f"Prompt length: {len(long_text)}  -> Chunking...")
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
          print(f"   -> Attack found in chunk {i}-{i+len(chunk_text)}! , score = {score_output}")
          return is_safe, result_msg

      return True, "Guard Layer 2 : All chunks are Safe."