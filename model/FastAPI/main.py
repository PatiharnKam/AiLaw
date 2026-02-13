import sys
import os
import json

current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.append(current_dir)

from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import List, Optional, AsyncGenerator
import uvicorn

# Original agents
from agents.guard_agent import GuardAgent
from agents.detail_agent import DetailsAgent

# Streaming agents
from agents.detail_agent_streaming import DetailsAgentStreaming

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

# Original agents
guard_agent = GuardAgent()
detail_agent = DetailsAgent()

# Streaming agents
detail_agent_streaming = DetailsAgentStreaming()

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
            "totalInputTokens": detail_response.get("totalInputTokens", {}),
            "totalOutputTokens": detail_response.get("totalOutputTokens", {}),
            "finalOutputTokens": detail_response.get("finalOutputTokens", {}),
            "totalUsedTokens": detail_response.get("totalUsedTokens", {})
        }

    except Exception as e:
        import traceback
        error_detail = traceback.format_exc()
        print(f"Server Error Detail:\n{error_detail}")
        raise HTTPException(status_code=500, detail=str(e))
    
@app.post("/v1/chat/cot")
async def chat_completions_cot(request: ChatRequest):
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
        detail_response = await detail_agent.get_response_COT(user_question)
        
        return {
            "role": "assistant",
            "content": detail_response["content"],
            "decision": "processed",
            "memory": detail_response.get("memory", {}),
            "totalInputTokens": detail_response.get("totalInputTokens", {}),
            "totalOutputTokens": detail_response.get("totalOutputTokens", {}),
            "finalOutputTokens": detail_response.get("finalOutputTokens", {}),
            "totalUsedTokens": detail_response.get("totalUsedTokens", {})
        }

    except Exception as e:
        import traceback
        error_detail = traceback.format_exc()
        print(f"Server Error Detail:\n{error_detail}")
        raise HTTPException(status_code=500, detail=str(e))


# ---------------------------------------------------------
# API ROUTES - Streaming
# ---------------------------------------------------------

@app.post("/v1/chat/stream")
async def chat_completions_stream(request: ChatRequest):
    """
    Streaming non-COT endpoint
    Returns SSE stream
    """
    input_messages = [msg.model_dump() for msg in request.messages]
    
    return StreamingResponse(
        generate_sse_stream(input_messages, use_cot=False),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",  # Disable nginx buffering
        }
    )


@app.post("/v1/chat/cot/stream")
async def chat_completions_cot_stream(request: ChatRequest):
    """
    Streaming COT endpoint
    Returns SSE stream with plan steps and final answer
    """
    input_messages = [msg.model_dump() for msg in request.messages]
    
    return StreamingResponse(
        generate_sse_stream(input_messages, use_cot=True),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        }
    )

# ---------------------------------------------------------
# Helper: SSE Generator
# ---------------------------------------------------------
async def generate_sse_stream(
    messages: List[dict],
    use_cot: bool = False
) -> AsyncGenerator[str, None]:
    """
    Generate SSE stream for chat response
    """
    try:
        guard_response = await guard_agent.get_response(messages)
        
        decision = guard_response.get("memory", {}).get("guard_decision", "unknown")
        if "not allowed" in decision:
            content = guard_response.get("content", "ไม่สามารถตอบคำถามนี้ได้")
            
            yield f"data: {json.dumps({'type': 'content', 'text': content})}\n\n"
            yield f"data: {json.dumps({'type': 'done', 'input_tokens': 0, 'output_tokens': 0, 'total_tokens': 0, 'full_content': content, 'guard_blocked': True})}\n\n"
            yield "data: [DONE]\n\n"
            return

        # Step 2: Send guard passed event
        yield f"data: {json.dumps({'type': 'guard_passed'})}\n\n"

        # Step 3: Get user question
        user_question = messages[-1]['content']

        # Step 4: Stream response from detail agent
        if use_cot:
            stream_generator = detail_agent_streaming.get_response_COT_stream(user_question)
        else:
            stream_generator = detail_agent_streaming.get_response_non_COT_stream(user_question)

        async for chunk in stream_generator:
            yield f"data: {json.dumps(chunk)}\n\n"

        yield "data: [DONE]\n\n"

    except Exception as e:
        import traceback
        error_detail = traceback.format_exc()
        print(f"Streaming Error:\n{error_detail}")
        yield f"data: {json.dumps({'type': 'error', 'error': str(e)})}\n\n"
        yield "data: [DONE]\n\n"



# ---------------------------------------------------------
# Health Check
# ---------------------------------------------------------

@app.get("/health")
def health_check():
    return {
        "status": "healthy",
        "agents": {
            "guard": "loaded",
            "detail": "loaded",
            "guard_streaming": "loaded",
            "detail_streaming": "loaded"
        }
    }


if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8000, loop="asyncio")