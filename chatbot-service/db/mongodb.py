"""
MongoDB database connection and collections.
"""
from pymongo import MongoClient
from pymongo.collection import Collection
from pymongo.database import Database
import logging

from config import settings

logger = logging.getLogger(__name__)

# MongoDB client (singleton)
_client: MongoClient = None
_db: Database = None


def get_client() -> MongoClient:
    """Get or create MongoDB client."""
    global _client
    if _client is None:
        logger.info("Connecting to MongoDB...")
        _client = MongoClient(settings.MONGO_URI)
        # Test connection
        _client.admin.command('ping')
        logger.info("Successfully connected to MongoDB")
    return _client


def get_database() -> Database:
    """Get the database instance."""
    global _db
    if _db is None:
        client = get_client()
        _db = client[settings.DB_NAME]
        logger.info(f"Using database: {settings.DB_NAME}")
    return _db


def get_collection(name: str) -> Collection:
    """Get a collection by name."""
    db = get_database()
    return db[name]


# Collection accessors
def get_materials_collection() -> Collection:
    """Get the materials collection."""
    return get_collection("materials")


def get_suppliers_collection() -> Collection:
    """Get the suppliers collection."""
    return get_collection("suppliers")


def get_professionals_collection() -> Collection:
    """Get the professionals collection."""
    return get_collection("professionals")


def get_items_collection() -> Collection:
    """Get the items collection."""
    return get_collection("items")


def get_types_collection() -> Collection:
    """Get the types collection."""
    return get_collection("types")


def close_connection():
    """Close the MongoDB connection."""
    global _client, _db
    if _client is not None:
        _client.close()
        _client = None
        _db = None
        logger.info("MongoDB connection closed")

def test_connection() -> bool:
    """
    Test the MongoDB connection.
    
    Returns:
        True if connection successful, False otherwise
    """
    try:
        client = get_client()
        client.admin.command('ping')
        logger.info("MongoDB connection test successful")
        return True
    except Exception as e:
        logger.error(f"MongoDB connection test failed: {e}")
        return False