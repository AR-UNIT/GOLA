Image Sharing platform, allowing users to:
 - upload, download, share images
 - tag images with keywords
 - capture user interactions/events with images (likes, dislikes, comments, views)
 - user management(authentication)
 - search images

Implemented using 
 - Kafka as a message broker, 
 - MinIO/S3 as the image store
 - PostgresDB as the table/metadata store
 - Redis as the caching layer
 - CRON jobs for durable persistence of events/user interactions to PostgresDB
 - RateLimiting for API
 - elastic search for searching images via title/tags (WIP)


### HIGH LEVEL SOFTWARE ARCHITECTURE DIAGRAM
![high level design diagram](./high%20level%20design%20diagram.png)