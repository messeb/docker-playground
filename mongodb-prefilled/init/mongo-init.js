const fs = require('fs');

const databaseName = 'projectdb';
const collectionName = 'project';
const userName = 'myuser';
const password = 'mypassword';

// Content should be an array of similar objects
const data = fs.readFileSync('/docker-entrypoint-initdb.d/data.json');
const jsonData = JSON.parse(data);

// Connect to the database
conn = new Mongo();
db = conn.getDB(databaseName);

// Create a new user
db.createUser({
    user: userName,
    pwd: password,
    roles: [
        {  
            role: "readWrite", 
            db: databaseName 
        }
    ]
});

// Creates an index on the fieldname
// Documentation: https://www.mongodb.com/docs/manual/reference/method/db.collection.createIndex/
/*
db[collectionName].createIndex({ "fieldname": 1 });
*/

// Insert data
db[collectionName].insertMany(jsonData);
