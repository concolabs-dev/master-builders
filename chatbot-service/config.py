import os
from dotenv import load_dotenv
from pydantic_settings import BaseSettings

load_dotenv()


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""
    
    # MongoDB
    MONGO_URI: str = os.getenv("MONGO_URI", "")
    DB_NAME: str = os.getenv("DB_NAME", "materials_db")
    
    # Google Gemini
    GOOGLE_API_KEY: str = os.getenv("GOOGLE_API_KEY", "")
    GEMINI_MODEL: str = os.getenv("GEMINI_MODEL", "gemini-2.5-flash")
    
    # Service configuration
    CHATBOT_PORT: int = int(os.getenv("CHATBOT_PORT", "8050"))
    CHATBOT_HOST: str = os.getenv("CHATBOT_HOST", "0.0.0.0")
    DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"
    
    # API Security (optional)
    API_KEY: str = os.getenv("CHATBOT_API_KEY", "")
    
    class Config:
        env_file = ".env"
        extra = "ignore"


settings = Settings()
