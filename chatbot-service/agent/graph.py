"""
LangGraph agent with Google Gemini integration.
"""
from typing import TypedDict, Annotated, Sequence, Literal
import operator
import logging
import asyncio

from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, SystemMessage
from langchain_google_genai import ChatGoogleGenerativeAI
from langgraph.graph import StateGraph, END
from langgraph.prebuilt import ToolNode
from langgraph.checkpoint.memory import MemorySaver

from agent.tools import all_tools
from agent.prompts import SYSTEM_PROMPT
from config import settings

logger = logging.getLogger(__name__)


# Define the state
class AgentState(TypedDict):
    """State for the agent graph."""
    messages: Annotated[Sequence[BaseMessage], operator.add]
    session_id: str


def get_llm():
    """
    Get the LLM (Gemini) with tools bound.
    
    Returns:
        Configured Gemini LLM with tools
    """
    model_name = settings.GEMINI_MODEL
    
    # Try different model name formats
    llm = ChatGoogleGenerativeAI(
        model=model_name,
        google_api_key=settings.GOOGLE_API_KEY,
        temperature=0.2,  # Slightly higher for more natural responses
        convert_system_message_to_human=True,
        max_retries=3,  # Add retries
    )
    
    return llm.bind_tools(all_tools)


def agent_node(state: AgentState):
    """
    The agent node that processes messages and decides actions.
    """
    logger.info(f"Agent processing {len(state['messages'])} messages")
    
    try:
        llm = get_llm()
        
        # Prepend system message if not present
        messages = list(state["messages"])
        if not messages or not isinstance(messages[0], SystemMessage):
            messages = [SystemMessage(content=SYSTEM_PROMPT)] + messages
        
        response = llm.invoke(messages)
        return {"messages": [response]}
        
    except Exception as e:
        logger.error(f"Agent error: {e}")
        # Return helpful error message
        error_msg = AIMessage(
            content="I apologize, but I encountered an error processing your request. Please try again or rephrase your question. You can ask me about:\n- Material prices (cement, sand, electrical items)\n- Suppliers and their products\n- Construction professionals"
        )
        return {"messages": [error_msg]}


def should_continue(state: AgentState) -> Literal["tools", "end"]:
    """
    Determine if we should continue to tools or end.
    """
    last_message = state["messages"][-1]
    
    # If there are tool calls, continue to tools
    if hasattr(last_message, "tool_calls") and last_message.tool_calls:
        logger.info(f"Routing to tools - {len(last_message.tool_calls)} tool calls")
        return "tools"
    
    # Otherwise, end
    logger.info("Routing to end - no tool calls")
    return "end"


def create_agent_graph():
    """
    Create and compile the LangGraph agent.
    """
    logger.info("Creating agent graph...")
    
    workflow = StateGraph(AgentState)
    
    workflow.add_node("agent", agent_node)
    workflow.add_node("tools", ToolNode(all_tools))
    
    workflow.set_entry_point("agent")
    
    workflow.add_conditional_edges(
        "agent",
        should_continue,
        {
            "tools": "tools",
            "end": END
        }
    )
    
    workflow.add_edge("tools", "agent")
    
    memory = MemorySaver()
    app = workflow.compile(checkpointer=memory)
    
    logger.info("Agent graph created successfully")
    return app


async def chat(message: str, session_id: str, history: list = None) -> str:
    """
    Process a chat message and return the response.
    """
    logger.info(f"Chat called - session: {session_id}, message: {message[:50]}...")
    
    try:
        agent_graph = create_agent_graph()
        
        messages = []
        
        if history:
            for msg in history:
                if msg["role"] == "user":
                    messages.append(HumanMessage(content=msg["content"]))
                elif msg["role"] == "assistant":
                    messages.append(AIMessage(content=msg["content"]))
        
        messages.append(HumanMessage(content=message))
        
        config = {"configurable": {"thread_id": session_id}}
        
        result = await agent_graph.ainvoke(
            {"messages": messages, "session_id": session_id},
            config
        )
        
        final_message = result["messages"][-1]
        
        logger.info(f"Chat completed - response length: {len(final_message.content)}")
        return final_message.content
        
    except Exception as e:
        logger.error(f"Chat processing error: {e}")
        return f"I apologize, but I'm having trouble processing your request right now. Please try again in a moment.\n\nYou can ask me about:\n- Material prices\n- Suppliers\n- Construction professionals\n\nError details: {str(e)[:100]}"

def get_agent_graph():
    """Alias for create_agent_graph for backward compatibility."""
    return create_agent_graph()
def chat_sync(message: str, session_id: str, history: list = None) -> str:
    """
    Process a chat message and return the response (sync version).
    Wrapper around the async chat function.
    """
    return asyncio.run(chat(message, session_id, history))