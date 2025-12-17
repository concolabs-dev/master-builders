"""
FastAPI main application for the BuildMarket Chatbot Service.
"""
from datetime import datetime
from typing import Optional, List
from contextlib import asynccontextmanager
import logging
import uuid

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from agent import chat, GREETING_MESSAGE
from agent.prompts import GREETING_SUGGESTIONS
from db import test_connection

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    logger.info("Starting BuildMarket Chatbot Service...")
    
    # Test database connection
    if test_connection():
        logger.info("Database connection successful")
    else:
        logger.warning("Database connection failed - some features may not work")
    
    yield
    
    logger.info("Shutting down BuildMarket Chatbot Service...")


app = FastAPI(
    title="BuildMarket Chatbot API",
    description="AI-powered chatbot for construction materials marketplace",
    version="1.0.0",
    lifespan=lifespan
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Request/Response models
class ChatMessage(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    message: str
    session_id: Optional[str] = None
    history: Optional[List[ChatMessage]] = None


class ChatResponse(BaseModel):
    response: str
    session_id: str
    timestamp: str


class GreetingResponse(BaseModel):
    message: str
    session_id: str
    suggestions: List[str]


class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
    database: str
    timestamp: str


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    db_status = "connected" if test_connection() else "disconnected"
    
    return HealthResponse(
        status="healthy" if db_status == "connected" else "degraded",
        service="buildmarket-chatbot",
        version="1.0.0",
        database=db_status,
        timestamp=datetime.utcnow().isoformat()
    )


@app.get("/greeting", response_model=GreetingResponse)
async def get_greeting():
    """Get initial greeting message with suggestions."""
    session_id = f"session_{uuid.uuid4().hex[:12]}"
    
    return GreetingResponse(
        message=GREETING_MESSAGE,
        session_id=session_id,
        suggestions=GREETING_SUGGESTIONS
    )


@app.post("/chat", response_model=ChatResponse)
async def chat_endpoint(request: ChatRequest):
    """Process a chat message."""
    logger.info(f"Processing chat request - session: {request.session_id}, message: {request.message[:50]}...")
    
    # Generate session ID if not provided
    session_id = request.session_id or f"session_{uuid.uuid4().hex[:12]}"
    
    # Convert history to the expected format
    history = None
    if request.history:
        history = [{"role": msg.role, "content": msg.content} for msg in request.history]
    
    try:
        # Get response from agent
        response = await chat(
            message=request.message,
            session_id=session_id,
            history=history
        )
        
        logger.info(f"Chat response generated - session: {session_id}")
        
        return ChatResponse(
            response=response,
            session_id=session_id,
            timestamp=datetime.utcnow().isoformat()
        )
    
    except Exception as e:
        logger.error(f"Chat error: {str(e)}")
        return ChatResponse(
            response="I'm sorry, I encountered an error. Please try again with a simpler question like 'What is the price of cement?' or 'Show me professionals'.",
            session_id=session_id,
            timestamp=datetime.utcnow().isoformat()
        )


if __name__ == "__main__":
    import uvicorn
    from config import settings
    
    uvicorn.run(
        "main:app",
        host=settings.CHATBOT_HOST,
        port=settings.CHATBOT_PORT,
        reload=True
    )