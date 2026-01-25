1. Stop accepting new HTTP requests
2. Stop consuming new RabbitMQ messages
3. Wait for in-flight HTTP requests
4. Wait for in-flight RabbitMQ handlers
5. Stop background goroutines
6. Close DB pool
7. Close RabbitMQ channel
8. Close RabbitMQ connection
9. Exit process
