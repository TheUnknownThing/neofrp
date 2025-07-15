#!/usr/bin/env python3
"""
Simple TCP Server for testing FRP implementation
This server will receive forwarded connections from the FRP server
"""

import socket
import threading
import logging
import time
import signal
import sys
from datetime import datetime

# Configure detailed logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - [%(threadName)s] - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('/root/tcp_server.log')
    ]
)

logger = logging.getLogger(__name__)

class TCPServer:
    def __init__(self, host='0.0.0.0', port=25560):
        self.host = host
        self.port = port
        self.socket = None
        self.running = False
        self.client_count = 0
        
    def start(self):
        """Start the TCP server"""
        try:
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            self.socket.bind((self.host, self.port))
            self.socket.listen(5)
            self.running = True
            
            logger.info(f"TCP Server started on {self.host}:{self.port}")
            logger.info("Waiting for connections...")
            
            while self.running:
                try:
                    client_socket, client_address = self.socket.accept()
                    self.client_count += 1
                    client_thread = threading.Thread(
                        target=self.handle_client,
                        args=(client_socket, client_address, self.client_count),
                        name=f"Client-{self.client_count}"
                    )
                    client_thread.daemon = True
                    client_thread.start()
                    
                    logger.info(f"New connection #{self.client_count} from {client_address}")
                    
                except socket.error as e:
                    if self.running:
                        logger.error(f"Socket error: {e}")
                    break
                    
        except Exception as e:
            logger.error(f"Failed to start server: {e}")
        finally:
            self.stop()
    
    def handle_client(self, client_socket, client_address, client_id):
        """Handle individual client connections"""
        start_time = time.time()
        bytes_received = 0
        messages_received = 0
        
        try:
            logger.debug(f"Client #{client_id} handler started")
            
            while self.running:
                try:
                    # Receive data from client
                    data = client_socket.recv(1024)
                    if not data:
                        logger.info(f"Client #{client_id} disconnected gracefully")
                        break
                    
                    bytes_received += len(data)
                    messages_received += 1
                    
                    # Log received data
                    message = data.decode('utf-8', errors='replace')
                    logger.debug(f"Client #{client_id} sent: {message.strip()}")
                    logger.debug(f"Client #{client_id} data (hex): {data.hex()}")
                    
                    # Send response back to client
                    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    response = f"[{timestamp}] Server received: {message.strip()}\n"
                    client_socket.send(response.encode('utf-8'))
                    
                    logger.debug(f"Client #{client_id} response sent: {response.strip()}")
                    
                except socket.error as e:
                    logger.warning(f"Client #{client_id} socket error: {e}")
                    break
                except Exception as e:
                    logger.error(f"Client #{client_id} error: {e}")
                    break
        
        finally:
            duration = time.time() - start_time
            logger.info(f"Client #{client_id} session ended:")
            logger.info(f"  - Duration: {duration:.2f} seconds")
            logger.info(f"  - Messages received: {messages_received}")
            logger.info(f"  - Bytes received: {bytes_received}")
            
            try:
                client_socket.close()
                logger.debug(f"Client #{client_id} socket closed")
            except:
                pass
    
    def stop(self):
        """Stop the server"""
        logger.info("Stopping TCP server...")
        self.running = False
        if self.socket:
            try:
                self.socket.close()
                logger.info("Server socket closed")
            except:
                pass

def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info(f"Received signal {signum}, shutting down...")
    server.stop()
    sys.exit(0)

if __name__ == "__main__":
    # Set up signal handlers for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Create and start server
    server = TCPServer()
    
    try:
        logger.info("Starting TCP test server for FRP testing")
        server.start()
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
    finally:
        server.stop()
        logger.info("TCP server stopped")
