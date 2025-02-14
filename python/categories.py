from pymongo import MongoClient

# Connect to MongoDB
client = MongoClient("mongodb://localhost:27017/")
db = client["materials_db"]
materials_collection = db["materials"]  # Existing collection
types_collection = db["types"]  # Collection to store types

# Fetch all materials
materials = materials_collection.find()

# Dictionary to structure types data
types_data = {}

# Process each material document
for material in materials:
    type_name = material.get("Type")  # Sheet name
    category_name = material.get("Category", {}).get("Category")
    subcategory_name = material.get("Category", {}).get("Subcategory")
    sub_subcategory_name = material.get("Category", {}).get("Sub subcategory")

    if not type_name:
        continue  # Skip if type is missing

    # Create type entry if not exists
    if type_name not in types_data:
        types_data[type_name] = {"name": type_name, "categories": []}

    # Find or create category entry
    category_obj = next((c for c in types_data[type_name]["categories"] if c["name"] == category_name), None)
    if not category_obj:
        category_obj = {"name": category_name, "subcategories": []}
        types_data[type_name]["categories"].append(category_obj)

    # Find or create subcategory entry
    if subcategory_name:
        subcategory_obj = next((s for s in category_obj["subcategories"] if s["name"] == subcategory_name), None)
        if not subcategory_obj:
            subcategory_obj = {"name": subcategory_name, "sub_subcategories": []}
            category_obj["subcategories"].append(subcategory_obj)

        # Add sub-subcategory
        if sub_subcategory_name and sub_subcategory_name not in subcategory_obj["sub_subcategories"]:
            subcategory_obj["sub_subcategories"].append(sub_subcategory_name)

# Insert structured data into `types` collection
types_collection.delete_many({})  # Clear existing data
for type_entry in types_data.values():
    types_collection.insert_one(type_entry)

print("Types collection updated successfully!")
