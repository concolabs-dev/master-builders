"""
Agent module for BuildMarket chatbot.
"""
import asyncio

from agent.graph import chat, create_agent_graph, get_agent_graph
from agent.prompts import SYSTEM_PROMPT, GREETING_MESSAGE, GREETING_SUGGESTIONS


def chat_sync(message: str, session_id: str, history: list = None) -> str:
    """
    Process a chat message and return the response (sync version).
    """
    return asyncio.run(chat(message, session_id, history))


__all__ = [
    "chat",
    "chat_sync",
    "create_agent_graph",
    "get_agent_graph",
    "SYSTEM_PROMPT",
    "GREETING_MESSAGE",
    "GREETING_SUGGESTIONS"
]