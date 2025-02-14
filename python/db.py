import pandas as pd
from pymongo import MongoClient
import uuid
import re
# Load the Excel file
file_path = "./data/Material List VFORM (1).xlsx"
xls = pd.ExcelFile(file_path)

# Connect to MongoDB
client = MongoClient("mongodb://localhost:27017/")  # Change if needed
db = client["materials_db"]  # Database name
collection = db["materials"]  # Collection name
pattern = r'\d{4}-\d{2}-\d{2}'
# Iterate through each sheet
for sheet_name in xls.sheet_names:
 
    df = xls.parse(sheet_name, skiprows=1)  # Read sheet, skipping first two rows
    
    # Ensure correct column names
    df.columns = df.iloc[0].astype(str)
    df = df[1:].reset_index(drop=True)
    
    print(df.columns)
    # Extract relevant column names
    print(df.dtypes)
    #price_columns = [col for col in df.columns if isinstance(col, pd.Timestamp) or isinstance(col, str) and "-" in str(col)]
    price_columns = [col for col in df.columns if isinstance(col, str) and re.match(pattern, col)]

    print(price_columns)
    # Initialize category tracking
    category, subcategory, sub_subcategory = None, None, None
    print(df.head(10))
    
    
    # Iterate over rows
    for _, row in df.iterrows():
        if pd.isna(row["Nr"]):  # Category / Subcategory / SubSubCategory
            if not pd.isna(row["Materials list"]):
                if row["Materials list"].strip():  
                    if row["Materials list"] == row["Category"]:  
                        category = row["Materials list"]
                        subcategory, sub_subcategory = None, None
                    elif row["Materials list"] == row["SubCategory"]:
                        subcategory = row["Materials list"]
                        sub_subcategory = None
                    elif row["Materials list"] == row["SubSubCategory"]:
                        sub_subcategory = row["Materials list"]
        else:  # This is an item
            # print(row)
            # print( [[col, row[col]] for col in price_columns ])
            # print([{col: row[col] if pd.notna(row[col]) else None} for col in price_columns])
            item_data = {
                "Number": str(uuid.uuid4()),  # Generate unique ID
                "Name": row["Materials list"],
                "Type": sheet_name,
                "Category": {
                    "Category": row["Category"] if not pd.isna(row["Category"]) else None,
                    "Subcategory": row["SubCategory"] if not pd.isna(row["SubCategory"]) else None,
                    "Sub subcategory": row["SubSubCategory"] if not pd.isna(row["SubSubCategory"]) else None
                },
                "Qty": int(row["QTY"]) if not pd.isna(row["QTY"]) else None,
                "Unit": row["Unit"],
                "Prices":   [[col, row[col] if pd.notna(row[col]) else None] for col in price_columns],

                "Source": row["Source"] if "Source" in row else None
            }
            #[[col, row[col] if pd.notna(row[col]) else None] for col in price_columns],
            # Insert into MongoDB
            collection.insert_one(item_data)
    

print("Data successfully inserted into MongoDB!")
