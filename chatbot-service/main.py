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
from config import settings
from google import genai as google_genai

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def check_api_key_validity():
    """Check if the Google Gemini API key is valid and has credits."""
    try:
        if not settings.GOOGLE_API_KEY:
            return False, "API key not configured"
        
        # Try a simple API call to validate
        client = google_genai.Client(api_key=settings.GOOGLE_API_KEY)
        
        # List models as a simple validation
        models = client.models.list()
        
        if models:
            return True, "API key valid"
        else:
            return False, "No models available"
            
    except Exception as e:
        error_msg = str(e).lower()
        if "quota" in error_msg or "limit" in error_msg or "429" in str(e):
            return False, "API quota exceeded"
        elif "invalid" in error_msg or "unauthorized" in error_msg or "403" in str(e) or "401" in str(e):
            return False, "Invalid API key"
        else:
            return False, f"API error: {str(e)[:50]}"


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
    api_key: str
    api_status: str
    timestamp: str


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    db_status = "connected" if test_connection() else "disconnected"
    
    # Check API key validity
    api_valid, api_message = check_api_key_validity()
    api_key_status = "valid" if api_valid else "invalid"
    
    # Determine overall status
    overall_status = "healthy" if (db_status == "connected" and api_valid) else "degraded"
    
    return HealthResponse(
        status=overall_status,
        service="buildmarket-chatbot",
        version="1.0.0",
        database=db_status,
        api_key=api_key_status,
        api_status=api_message,
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