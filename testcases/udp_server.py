#!/usr/bin/env python3
"""
Simple UDP Server for testing FRP implementation
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
        logging.FileHandler('/root/udp_server.log')
    ]
)

logger = logging.getLogger(__name__)

class UDPServer:
    def __init__(self, host='0.0.0.0', port=25561):
        self.host = host
        self.port = port
        self.socket = None
        self.running = False
        self.message_count = 0
        self.clients = {}  # Track clients by address
        
    def start(self):
        """Start the UDP server"""
        try:
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            self.socket.bind((self.host, self.port))
            self.running = True
            
            logger.info(f"UDP Server started on {self.host}:{self.port}")
            logger.info("Waiting for messages...")
            
            while self.running:
                try:
                    # Receive data from client
                    data, client_address = self.socket.recvfrom(4096)
                    self.message_count += 1
                    
                    # Track new clients
                    if client_address not in self.clients:
                        self.clients[client_address] = {
                            'first_seen': datetime.now(),
                            'message_count': 0
                        }
                        logger.info(f"New client connected: {client_address}")
                    
                    self.clients[client_address]['message_count'] += 1
                    
                    # Decode and log the message
                    try:
                        message = data.decode('utf-8')
                        logger.info(f"Received from {client_address}: {message}")
                        
                        # Echo the message back
                        response = f"Echo: {message} (msg #{self.message_count})"
                        self.socket.sendto(response.encode('utf-8'), client_address)
                        logger.info(f"Sent response to {client_address}: {response}")
                        
                    except UnicodeDecodeError:
                        logger.warning(f"Received non-UTF-8 data from {client_address}: {data}")
                        response = f"Binary data received ({len(data)} bytes)"
                        self.socket.sendto(response.encode('utf-8'), client_address)
                        
                except socket.timeout:
                    continue
                except Exception as e:
                    if self.running:
                        logger.error(f"Error handling client: {e}")
                    
        except Exception as e:
            logger.error(f"Server error: {e}")
        finally:
            self.stop()
            
    def stop(self):
        """Stop the UDP server"""
        if self.running:
            self.running = False
            logger.info("Stopping UDP server...")
            
            if self.socket:
                self.socket.close()
                
            logger.info(f"Server statistics:")
            logger.info(f"Total messages received: {self.message_count}")
            logger.info(f"Total clients: {len(self.clients)}")
            
            for addr, info in self.clients.items():
                logger.info(f"Client {addr}: {info['message_count']} messages, "
                           f"connected at {info['first_seen']}")

# Global server instance for signal handling
server_instance = None

def signal_handler(signum, frame):
    logger.info(f"Received signal {signum}")
    if server_instance:
        server_instance.stop()
    sys.exit(0)

if __name__ == "__main__":
    # Set up signal handlers for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Create and start server
    server_instance = UDPServer()
    
    try:
        logger.info("Starting UDP test server for FRP testing")
        server_instance.start()
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
    finally:
        if server_instance:
            server_instance.stop()
        logger.info("UDP server stopped")
