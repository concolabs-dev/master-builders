"""
LangGraph tools for querying the BuildMarket database.
These tools allow the AI agent to search for materials, suppliers, and professionals.
"""
from langchain_core.tools import tool
from typing import Optional, List, Dict, Any
import logging
import re

from db import (
    get_materials_collection,
    get_suppliers_collection,
    get_professionals_collection,
    get_items_collection,
    get_types_collection
)

logger = logging.getLogger(__name__)


# Common search term mappings for better matching
MATERIAL_SYNONYMS = {
    "earth cable": ["earth", "earthing", "grounding", "cable", "wire", "electrical"],
    "electrical cable": ["cable", "wire", "wires", "electrical", "conductor"],
    "cement": ["cement", "portland", "opc", "ppc"],
    "sand": ["sand", "river sand", "sea sand", "m-sand"],
    "metal": ["metal", "aggregate", "gravel", "stone"],
    "brick": ["brick", "block", "cement block", "clay brick"],
    "steel": ["steel", "rebar", "reinforcement", "bar", "rod", "tmt"],
    "pipe": ["pipe", "pvc", "upvc", "cpvc", "conduit"],
    "paint": ["paint", "emulsion", "enamel", "primer", "coating"],
    "tile": ["tile", "ceramic", "porcelain", "floor tile", "wall tile"],
    "wood": ["wood", "timber", "plywood", "lumber"],
    "roof": ["roof", "roofing", "sheet", "zinc", "asbestos"],
}

def expand_search_terms(query: str) -> List[str]:
    """Expand search query with synonyms and related terms."""
    query_lower = query.lower().strip()
    terms = [query]
    
    # Check for synonym matches
    for key, synonyms in MATERIAL_SYNONYMS.items():
        if any(word in query_lower for word in key.split()):
            terms.extend(synonyms)
    
    # Add individual words from the query
    words = query_lower.split()
    terms.extend(words)
    
    # Remove duplicates while preserving order
    seen = set()
    unique_terms = []
    for t in terms:
        if t.lower() not in seen:
            seen.add(t.lower())
            unique_terms.append(t)
    
    return unique_terms


@tool
def search_materials(
    query: str,
    category: Optional[str] = None,
    subcategory: Optional[str] = None,
    type_name: Optional[str] = None
) -> List[Dict[str, Any]]:
    """
    Search for construction materials by name, category, subcategory, or type.
    Use this tool when users ask about material prices, availability, or specifications.
    
    Args:
        query: Search term (material name or keyword like "cement", "sand", "metal", "brick", "cable", "wire")
        category: Optional category filter (e.g., "RAW MATERIAL", "CONCRETE", "BRICK & BLOCK", "WIRES")
        subcategory: Optional subcategory filter
        type_name: Optional type filter (e.g., "Civil Items", "Electrical Items", "Plumbing Items")
    
    Returns:
        List of materials with their details and latest prices
    """
    logger.info(f"search_materials called: query='{query}', category='{category}', subcategory='{subcategory}', type='{type_name}'")
    
    collection = get_materials_collection()
    
    # Expand search terms for better matching
    search_terms = expand_search_terms(query)
    logger.info(f"Expanded search terms: {search_terms}")
    
    # Build OR conditions for each search term
    or_conditions = []
    for term in search_terms:
        or_conditions.extend([
            {"Name": {"$regex": term, "$options": "i"}},
            {"Category.Category": {"$regex": term, "$options": "i"}},
            {"Category.Subcategory": {"$regex": term, "$options": "i"}},
            {"Category.Sub subcategory": {"$regex": term, "$options": "i"}},
            {"Type": {"$regex": term, "$options": "i"}},
        ])
    
    filter_query = {"$or": or_conditions}
    
    # Add specific filters if provided
    if category:
        filter_query["Category.Category"] = {"$regex": category, "$options": "i"}
    if subcategory:
        filter_query["Category.Subcategory"] = {"$regex": subcategory, "$options": "i"}
    if type_name:
        filter_query["Type"] = {"$regex": type_name, "$options": "i"}
    
    materials = list(collection.find(filter_query).limit(15))
    
    # If no results, try a more relaxed search
    if not materials:
        logger.info("No results with strict search, trying relaxed search")
        # Search with just the first word
        first_word = query.split()[0] if query.split() else query
        relaxed_filter = {
            "$or": [
                {"Name": {"$regex": first_word, "$options": "i"}},
                {"Category.Category": {"$regex": first_word, "$options": "i"}},
                {"Category.Subcategory": {"$regex": first_word, "$options": "i"}},
                {"Type": {"$regex": first_word, "$options": "i"}},
            ]
        }
        materials = list(collection.find(relaxed_filter).limit(15))
    
    # Format response
    results = []
    for mat in materials:
        # Get latest price
        latest_price = None
        price_date = None
        if mat.get("Prices") and len(mat["Prices"]) > 0:
            # Find the latest non-null price
            for price_entry in reversed(mat["Prices"]):
                if len(price_entry) >= 2 and price_entry[1] is not None:
                    price_date = price_entry[0]
                    latest_price = price_entry[1]
                    break
        
        results.append({
            "id": str(mat.get("_id")),
            "name": mat.get("Name"),
            "type": mat.get("Type"),
            "category": mat.get("Category", {}).get("Category"),
            "subcategory": mat.get("Category", {}).get("Subcategory"),
            "sub_subcategory": mat.get("Category", {}).get("Sub subcategory"),
            "unit": mat.get("Unit"),
            "latest_price": latest_price,
            "price_date": price_date,
        })
    
    logger.info(f"search_materials found {len(results)} materials")
    
    if not results:
        return [{
            "message": f"No materials found matching '{query}'.",
            "suggestion": "Try searching by category like 'WIRES' for electrical items, 'RAW MATERIAL' for cement/sand, or 'CONCRETE' for concrete materials.",
            "available_types": ["Civil Items", "Electrical Items", "Plumbing Items"]
        }]
    
    return results


@tool
def search_electrical_materials(query: Optional[str] = None) -> List[Dict[str, Any]]:
    """
    Search specifically for electrical materials like wires, cables, switches, lights, etc.
    Use this when users ask about electrical items, cables, wires, or any electrical materials.
    
    Args:
        query: Optional search term to filter electrical materials (e.g., "cable", "wire", "switch", "light")
    
    Returns:
        List of electrical materials with their details and prices
    """
    logger.info(f"search_electrical_materials called: query='{query}'")
    
    collection = get_materials_collection()
    
    # Base filter for electrical items
    filter_query = {"Type": {"$regex": "Electrical", "$options": "i"}}
    
    # Add query filter if provided
    if query:
        search_terms = expand_search_terms(query)
        or_conditions = []
        for term in search_terms:
            or_conditions.extend([
                {"Name": {"$regex": term, "$options": "i"}},
                {"Category.Category": {"$regex": term, "$options": "i"}},
                {"Category.Subcategory": {"$regex": term, "$options": "i"}},
            ])
        filter_query["$or"] = or_conditions
    
    materials = list(collection.find(filter_query).limit(20))
    
    results = []
    for mat in materials:
        latest_price = None
        price_date = None
        if mat.get("Prices") and len(mat["Prices"]) > 0:
            for price_entry in reversed(mat["Prices"]):
                if len(price_entry) >= 2 and price_entry[1] is not None:
                    price_date = price_entry[0]
                    latest_price = price_entry[1]
                    break
        
        results.append({
            "id": str(mat.get("_id")),
            "name": mat.get("Name"),
            "category": mat.get("Category", {}).get("Category"),
            "subcategory": mat.get("Category", {}).get("Subcategory"),
            "unit": mat.get("Unit"),
            "latest_price": latest_price,
            "price_date": price_date,
        })
    
    logger.info(f"search_electrical_materials found {len(results)} materials")
    
    if not results:
        return [{"message": "No electrical materials found. The database may not have electrical items yet."}]
    
    return results


@tool
def search_plumbing_materials(query: Optional[str] = None) -> List[Dict[str, Any]]:
    """
    Search specifically for plumbing materials like pipes, fittings, drainage items, etc.
    Use this when users ask about pipes, plumbing, drainage, or water-related materials.
    
    Args:
        query: Optional search term to filter plumbing materials (e.g., "pipe", "fitting", "drain")
    
    Returns:
        List of plumbing materials with their details and prices
    """
    logger.info(f"search_plumbing_materials called: query='{query}'")
    
    collection = get_materials_collection()
    
    # Base filter for plumbing items
    filter_query = {"Type": {"$regex": "Plumbing", "$options": "i"}}
    
    if query:
        search_terms = expand_search_terms(query)
        or_conditions = []
        for term in search_terms:
            or_conditions.extend([
                {"Name": {"$regex": term, "$options": "i"}},
                {"Category.Category": {"$regex": term, "$options": "i"}},
                {"Category.Subcategory": {"$regex": term, "$options": "i"}},
            ])
        filter_query["$or"] = or_conditions
    
    materials = list(collection.find(filter_query).limit(20))
    
    results = []
    for mat in materials:
        latest_price = None
        price_date = None
        if mat.get("Prices") and len(mat["Prices"]) > 0:
            for price_entry in reversed(mat["Prices"]):
                if len(price_entry) >= 2 and price_entry[1] is not None:
                    price_date = price_entry[0]
                    latest_price = price_entry[1]
                    break
        
        results.append({
            "id": str(mat.get("_id")),
            "name": mat.get("Name"),
            "category": mat.get("Category", {}).get("Category"),
            "subcategory": mat.get("Category", {}).get("Subcategory"),
            "unit": mat.get("Unit"),
            "latest_price": latest_price,
            "price_date": price_date,
        })
    
    logger.info(f"search_plumbing_materials found {len(results)} materials")
    
    if not results:
        return [{"message": "No plumbing materials found."}]
    
    return results


@tool
def get_material_price_history(material_name: str) -> Dict[str, Any]:
    """
    Get the price history for a specific material.
    Use this when users want to see price trends or historical prices.
    
    Args:
        material_name: The name of the material (e.g., "Cement", "Sand", "Metal")
    
    Returns:
        Material details with full price history showing how prices changed over time
    """
    logger.info(f"get_material_price_history called for: '{material_name}'")
    
    collection = get_materials_collection()
    
    # Try exact match first
    material = collection.find_one(
        {"Name": {"$regex": f"^{material_name}$", "$options": "i"}}
    )
    
    # If not found, try partial match
    if not material:
        material = collection.find_one(
            {"Name": {"$regex": material_name, "$options": "i"}}
        )
    
    if not material:
        # Try to find similar materials
        similar = list(collection.find(
            {"Name": {"$regex": material_name.split()[0], "$options": "i"}}
        ).limit(5))
        
        suggestions = [m.get("Name") for m in similar] if similar else []
        
        return {
            "error": f"Material '{material_name}' not found.",
            "suggestions": suggestions if suggestions else "Try searching for materials first to see what's available."
        }
    
    # Format price history
    price_history = []
    if material.get("Prices"):
        for entry in material["Prices"]:
            if len(entry) >= 2:
                price_history.append({
                    "date": entry[0],
                    "price": entry[1]
                })
    
    # Get prices with values only
    prices_with_values = [p for p in price_history if p["price"] is not None]
    
    return {
        "name": material.get("Name"),
        "unit": material.get("Unit"),
        "type": material.get("Type"),
        "category": material.get("Category", {}).get("Category"),
        "subcategory": material.get("Category", {}).get("Subcategory"),
        "price_history": price_history[-12:],  # Last 12 months
        "total_records": len(price_history),
        "records_with_prices": len(prices_with_values)
    }


@tool
def search_suppliers(
    query: Optional[str] = None,
    location: Optional[str] = None
) -> List[Dict[str, Any]]:
    """
    Search for approved suppliers by business name or location.
    Use this when users ask about suppliers, vendors, or where to buy materials.
    
    Args:
        query: Search term (business name or keyword)
        location: Optional location/address filter (e.g., "Colombo", "Kandy")
    
    Returns:
        List of approved suppliers with their contact details
    """
    logger.info(f"search_suppliers called: query='{query}', location='{location}'")
    
    collection = get_suppliers_collection()
    
    filter_query = {"status": "approved"}
    
    or_conditions = []
    
    if query:
        or_conditions.extend([
            {"business_name": {"$regex": query, "$options": "i"}},
            {"business_description": {"$regex": query, "$options": "i"}},
        ])
    
    if location:
        or_conditions.extend([
            {"address": {"$regex": location, "$options": "i"}},
            {"location": {"$regex": location, "$options": "i"}},
        ])
    
    if or_conditions:
        filter_query["$or"] = or_conditions
    
    suppliers = list(collection.find(filter_query).limit(10))
    
    results = []
    for sup in suppliers:
        results.append({
            "id": str(sup.get("_id")),
            "business_name": sup.get("business_name"),
            "description": sup.get("business_description"),
            "telephone": sup.get("telephone"),
            "email": sup.get("email_given"),
            "address": sup.get("address"),
            "location": sup.get("location")
        })
    
    logger.info(f"search_suppliers found {len(results)} suppliers")
    
    if not results:
        return [{"message": "No approved suppliers found matching your criteria. Try a different search term or location."}]
    
    return results


@tool
def search_professionals(
    query: Optional[str] = None,
    company_type: Optional[str] = None,
    specialization: Optional[str] = None
) -> List[Dict[str, Any]]:
    """
    Search for approved professionals (contractors, architects, engineers, quantity surveyors, etc.).
    Use this when users ask about construction professionals, contractors, or services.
    
    Args:
        query: Search term (company name, description, or service)
        company_type: Type of company (e.g., "Contractor", "Architect", "Engineer", "Interior Designer", "Quantity Surveyor")
        specialization: Area of specialization (e.g., "Residential", "Commercial", "Industrial")
    
    Returns:
        List of approved professionals with their details and contact information
    """
    logger.info(f"search_professionals called: query='{query}', type='{company_type}', spec='{specialization}'")
    
    collection = get_professionals_collection()
    
    filter_query = {"status": "approved"}
    
    or_conditions = []
    
    if query:
        or_conditions.extend([
            {"company_name": {"$regex": query, "$options": "i"}},
            {"company_description": {"$regex": query, "$options": "i"}},
            {"services_offered": {"$regex": query, "$options": "i"}},
            {"specializations": {"$regex": query, "$options": "i"}},
            {"company_type": {"$regex": query, "$options": "i"}},
        ])
    
    if company_type:
        filter_query["company_type"] = {"$regex": company_type, "$options": "i"}
    
    if specialization:
        or_conditions.append({"specializations": {"$regex": specialization, "$options": "i"}})
    
    if or_conditions:
        filter_query["$or"] = or_conditions
    
    professionals = list(collection.find(filter_query).limit(10))
    
    results = []
    for prof in professionals:
        results.append({
            "id": str(prof.get("_id")),
            "company_name": prof.get("company_name"),
            "company_type": prof.get("company_type"),
            "description": prof.get("company_description"),
            "telephone": prof.get("telephone_number"),
            "email": prof.get("email"),
            "website": prof.get("website"),
            "address": prof.get("address"),
            "specializations": prof.get("specializations", []),
            "services_offered": prof.get("services_offered", []),
            "year_founded": prof.get("year_founded"),
            "employees": prof.get("number_of_employees")
        })
    
    logger.info(f"search_professionals found {len(results)} professionals")
    
    if not results:
        return [{
            "message": "No approved professionals found matching your criteria.",
            "suggestion": "Try searching without filters or use terms like 'contractor', 'architect', 'engineer'."
        }]
    
    return results


@tool
def get_top_professionals(company_type: Optional[str] = None, limit: int = 5) -> List[Dict[str, Any]]:
    """
    Get top-rated or most established professionals.
    Use this when users ask about "best", "top", or "recommended" professionals.
    
    Args:
        company_type: Optional filter by type (e.g., "Contractor", "Architect", "Engineer")
        limit: Number of results to return (default 5)
    
    Returns:
        List of top professionals sorted by experience (year founded)
    """
    logger.info(f"get_top_professionals called: type='{company_type}', limit={limit}")
    
    collection = get_professionals_collection()
    
    filter_query = {"status": "approved"}
    
    if company_type:
        filter_query["company_type"] = {"$regex": company_type, "$options": "i"}
    
    # Sort by year_founded (older = more established) and number of employees
    professionals = list(collection.find(filter_query).sort([
        ("year_founded", 1),  # Older companies first
        ("number_of_employees", -1)  # More employees = larger company
    ]).limit(limit))
    
    results = []
    for prof in professionals:
        results.append({
            "company_name": prof.get("company_name"),
            "company_type": prof.get("company_type"),
            "description": prof.get("company_description"),
            "year_founded": prof.get("year_founded"),
            "employees": prof.get("number_of_employees"),
            "telephone": prof.get("telephone_number"),
            "email": prof.get("email"),
            "website": prof.get("website"),
            "address": prof.get("address"),
            "specializations": prof.get("specializations", []),
        })
    
    logger.info(f"get_top_professionals found {len(results)} professionals")
    
    if not results:
        return [{"message": "No approved professionals found. Try searching with different criteria."}]
    
    return results


@tool
def get_supplier_items(supplier_name: str) -> Dict[str, Any]:
    """
    Get items/products offered by a specific supplier.
    Use this when users want to know what a supplier sells or their product catalog.
    
    Args:
        supplier_name: Name of the supplier
    
    Returns:
        List of items offered by the supplier with prices
    """
    logger.info(f"get_supplier_items called for: '{supplier_name}'")
    
    suppliers_collection = get_suppliers_collection()
    items_collection = get_items_collection()
    
    # First find the supplier
    supplier = suppliers_collection.find_one(
        {"business_name": {"$regex": supplier_name, "$options": "i"}}
    )
    
    if not supplier:
        # Try partial match
        supplier = suppliers_collection.find_one(
            {"business_name": {"$regex": supplier_name.split()[0], "$options": "i"}}
        )
    
    if not supplier:
        return {"error": f"Supplier '{supplier_name}' not found. Try searching for suppliers first."}
    
    # Get items for this supplier
    items = list(items_collection.find(
        {"supplierPid": supplier.get("pid"), "status": True}
    ).limit(20))
    
    results = []
    for item in items:
        results.append({
            "id": str(item.get("_id")),
            "name": item.get("name"),
            "description": item.get("description"),
            "category": item.get("category"),
            "subcategory": item.get("subcategory"),
            "price": item.get("price"),
            "unit": item.get("unit"),
            "image_url": item.get("imgUrl")
        })
    
    logger.info(f"get_supplier_items found {len(results)} items for supplier '{supplier_name}'")
    
    return {
        "supplier": supplier.get("business_name"),
        "supplier_address": supplier.get("address"),
        "supplier_phone": supplier.get("telephone"),
        "items_count": len(results),
        "items": results
    }


@tool
def get_material_categories() -> List[Dict[str, Any]]:
    """
    Get all available material categories and their structure.
    Use this when users want to browse categories or don't know what to search for.
    
    Returns:
        List of material types with their categories and subcategories
    """
    logger.info("get_material_categories called")
    
    collection = get_types_collection()
    
    types = list(collection.find({}))
    
    results = []
    for t in types:
        categories = []
        for cat in t.get("categories", []):
            subcategories = [sub.get("name") for sub in cat.get("subcategories", [])]
            categories.append({
                "name": cat.get("name"),
                "subcategories": subcategories if subcategories else None
            })
        
        results.append({
            "type_name": t.get("name"),
            "categories": categories
        })
    
    logger.info(f"get_material_categories found {len(results)} types")
    return results


@tool
def compare_material_prices(material_names: List[str]) -> Dict[str, Any]:
    """
    Compare prices of multiple materials.
    Use this when users want to compare costs of different materials.
    
    Args:
        material_names: List of material names to compare (e.g., ["Cement", "Sand", "Metal"])
    
    Returns:
        Comparison of the latest prices for each material with their units
    """
    logger.info(f"compare_material_prices called for: {material_names}")
    
    collection = get_materials_collection()
    
    comparisons = []
    
    for name in material_names[:5]:  # Limit to 5 materials
        material = collection.find_one(
            {"Name": {"$regex": name, "$options": "i"}}
        )
        
        if material:
            latest_price = None
            price_date = None
            if material.get("Prices"):
                for entry in reversed(material["Prices"]):
                    if len(entry) >= 2 and entry[1] is not None:
                        latest_price = entry[1]
                        price_date = entry[0]
                        break
            
            comparisons.append({
                "name": material.get("Name"),
                "unit": material.get("Unit"),
                "latest_price": latest_price,
                "price_date": price_date,
                "category": material.get("Category", {}).get("Category")
            })
        else:
            comparisons.append({
                "name": name,
                "error": "Not found"
            })
    
    logger.info(f"compare_material_prices compared {len(comparisons)} materials")
    
    return {
        "comparison": comparisons,
        "note": "Prices are in Sri Lankan Rupees (LKR) unless otherwise specified"
    }


@tool
def get_material_count_by_category() -> Dict[str, Any]:
    """
    Get the count of materials in each category.
    Use this to give users an overview of what's available in the database.
    
    Returns:
        Count of materials grouped by category
    """
    logger.info("get_material_count_by_category called")
    
    collection = get_materials_collection()
    
    pipeline = [
        {"$group": {"_id": "$Category.Category", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}},
        {"$limit": 20}
    ]
    
    results = list(collection.aggregate(pipeline))
    
    category_counts = []
    total = 0
    for r in results:
        if r["_id"]:
            category_counts.append({
                "category": r["_id"],
                "count": r["count"]
            })
            total += r["count"]
    
    logger.info(f"get_material_count_by_category found {len(category_counts)} categories")
    
    return {
        "total_materials": total,
        "categories": category_counts
    }


@tool
def get_database_overview() -> Dict[str, Any]:
    """
    Get an overview of all data available in the database.
    Use this when users ask general questions about what's available.
    
    Returns:
        Overview of materials, suppliers, and professionals counts
    """
    logger.info("get_database_overview called")
    
    materials_collection = get_materials_collection()
    suppliers_collection = get_suppliers_collection()
    professionals_collection = get_professionals_collection()
    
    # Count materials by type
    material_types = list(materials_collection.aggregate([
        {"$group": {"_id": "$Type", "count": {"$sum": 1}}}
    ]))
    
    # Count suppliers
    supplier_count = suppliers_collection.count_documents({"status": "approved"})
    
    # Count professionals by type
    professional_types = list(professionals_collection.aggregate([
        {"$match": {"status": "approved"}},
        {"$group": {"_id": "$company_type", "count": {"$sum": 1}}}
    ]))
    
    return {
        "materials": {
            "total": sum(t["count"] for t in material_types),
            "by_type": [{
                "type": t["_id"],
                "count": t["count"]
            } for t in material_types if t["_id"]]
        },
        "suppliers": {
            "total_approved": supplier_count
        },
        "professionals": {
            "total_approved": sum(p["count"] for p in professional_types),
            "by_type": [{
                "type": p["_id"],
                "count": p["count"]
            } for p in professional_types if p["_id"]]
        }
    }


# List of all tools for the agent
all_tools = [
    search_materials,
    search_electrical_materials,
    search_plumbing_materials,
    get_material_price_history,
    search_suppliers,
    search_professionals,
    get_top_professionals,
    get_supplier_items,
    get_material_categories,
    compare_material_prices,
    get_material_count_by_category,
    get_database_overview
]