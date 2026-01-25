import sys
import os

current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.append(current_dir)

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import uvicorn
from agents.guard_agent import GuardAgent
from agents.detail_agent import DetailsAgent

app = FastAPI(title="Legal Chatbot API")

# ---------------------------------------------------------
# Data Models
# ---------------------------------------------------------
class Message(BaseModel):
    role: str
    content: str

class ChatRequest(BaseModel):
    messages: List[Message]
    session_id: Optional[str] = None

# ---------------------------------------------------------
# Initialize Agents
# ---------------------------------------------------------
print("Loading Agents...")
guard_agent = GuardAgent()
detail_agent = DetailsAgent()
print("Agents Loaded Successfully!")

# ---------------------------------------------------------
# API ROUTES
# ---------------------------------------------------------

@app.get("/")
def read_root():
    return {"status": "online", "message": "Legal Chatbot API is running."}

@app.post("/v1/chat")
async def chat_completions(request: ChatRequest):
    try:
        input_messages = [msg.model_dump() for msg in request.messages]
        guard_response = await guard_agent.get_response(input_messages)
        
        # Guard Layer
        decision = guard_response.get("memory", {}).get("guard_decision", "unknown")
        if "not allowed" in decision:
            return {
                "role": "assistant",
                "content": guard_response["content"],
                "decision": decision
            }
        
        # Detail Layer
        user_question = input_messages[-1]['content']
        detail_response = await detail_agent.get_response_non_COT(user_question)
        
        return {
            "role": "assistant",
            "content": detail_response["content"],
            "decision": "processed",
            "memory": detail_response.get("memory", {}),
            "input_tokens" : detail_response.get("input_tokens", {}),
            "output_tokens" : detail_response.get("output_tokens", {}),
            "total_tokens" : detail_response.get("total_tokens", {})
        }

    except Exception as e:
        import traceback
        error_detail = traceback.format_exc()
        print(f"Server Error Detail:\n{error_detail}")
        raise HTTPException(status_code=500, detail=str(e))
    
@app.post("/v1/chat/cot")
async def chat_completions(request: ChatRequest):
    try:
        input_messages = [msg.model_dump() for msg in request.messages]
        guard_response = await guard_agent.get_response(input_messages)
        
        # Guard Layer
        decision = guard_response.get("memory", {}).get("guard_decision", "unknown")
        if "not allowed" in decision:
            return {
                "role": "assistant",
                "content": guard_response["content"],
                "decision": decision
            }
        
        # Detail Layer
        user_question = input_messages[-1]['content']
        detail_response = await detail_agent.get_response(user_question)
        
        return {
            "role": "assistant",
            "content": detail_response["content"],
            "decision": "processed",
            "memory": detail_response.get("memory", {}),
            "input_tokens" : detail_response.get("input_tokens", {}),
            "output_tokens" : detail_response.get("output_tokens", {}),
            "total_tokens" : detail_response.get("total_tokens", {})
        }

    except Exception as e:
        import traceback
        error_detail = traceback.format_exc()
        print(f"Server Error Detail:\n{error_detail}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8000, loop="asyncio")