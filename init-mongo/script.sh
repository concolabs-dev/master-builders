#!/bin/bash
echo "Starting MongoDB initialization script..."

# mongorestore will be run by the mongo container's entrypoint script
# which ensures mongod is ready.
# The dump is expected to be at /dump_data/materials_db inside the container.
# The target database is materials_db.
echo "Attempting to restore database 'materials_db' from '/dump_data/materials_db'..."
mongorestore --uri="mongodb://localhost:27017/materials_db" --dir="/dump_data/materials_db" --drop

if [ $? -eq 0 ]; then
  echo "Database restore completed successfully."
else
  echo "Database restore failed. Check mongorestore output."
fi

echo "MongoDB initialization script finished."
