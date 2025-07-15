#!/usr/bin/env python3
"""
Simple TCP Client for testing FRP implementation
This client will connect through the FRP tunnel to reach the server

Configuration:
- FRP Server: 10.26.116.54:3400 (QUIC protocol)
- TCP Service: Port 35560 (forwarded from client local port 25560)
- This client connects to the external FRP server port 35560
- The FRP client forwards local port 25560 to server port 35560
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
        # logging.FileHandler('/root/tcp_client.log')
    ]
)

logger = logging.getLogger(__name__)

class TCPClient:
    def __init__(self, server_host='10.26.116.54', server_port=35560):
        self.server_host = server_host
        self.server_port = server_port
        self.socket = None
        self.connected = False
        self.message_count = 0
        
    def connect(self):
        """Connect to the FRP server"""
        try:
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            logger.info(f"Attempting to connect to FRP server at {self.server_host}:{self.server_port}")
            
            start_time = time.time()
            self.socket.connect((self.server_host, self.server_port))
            connect_time = time.time() - start_time
            
            self.connected = True
            logger.info(f"Successfully connected to FRP server (took {connect_time:.3f}s)")
            logger.info(f"Local address: {self.socket.getsockname()}")
            logger.info(f"Remote address: {self.socket.getpeername()}")
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to connect to FRP server: {e}")
            self.connected = False
            return False
    
    def send_message(self, message):
        """Send a message to the server through FRP tunnel"""
        if not self.connected:
            logger.error("Not connected to server")
            return False
        
        try:
            self.message_count += 1
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            full_message = f"[{timestamp}] Client message #{self.message_count}: {message}"
            
            # Send message
            logger.debug(f"Sending message: {full_message}")
            data = full_message.encode('utf-8')
            bytes_sent = self.socket.send(data)
            logger.debug(f"Sent {bytes_sent} bytes, data (hex): {data.hex()}")
            
            # Receive response
            response = self.socket.recv(1024)
            if response:
                response_str = response.decode('utf-8', errors='replace')
                logger.info(f"Server response: {response_str.strip()}")
                logger.debug(f"Response data (hex): {response.hex()}")
                return True
            else:
                logger.warning("No response received from server")
                return False
                
        except Exception as e:
            logger.error(f"Failed to send message: {e}")
            self.connected = False
            return False
    
    def send_test_messages(self, count=5, interval=2):
        """Send multiple test messages"""
        logger.info(f"Starting to send {count} test messages with {interval}s interval")
        
        for i in range(count):
            if not self.connected:
                logger.error("Connection lost, stopping test")
                break
                
            message = f"Test message {i+1}/{count}"
            success = self.send_message(message)
            
            if not success:
                logger.error(f"Failed to send message {i+1}")
                break
            
            if i < count - 1:  # Don't wait after the last message
                logger.debug(f"Waiting {interval}s before next message...")
                time.sleep(interval)
        
        logger.info("Finished sending test messages")
    
    def disconnect(self):
        """Disconnect from the server"""
        logger.info("Disconnecting from server...")
        self.connected = False
        if self.socket:
            try:
                self.socket.close()
                logger.info("Client socket closed")
            except:
                pass
    
    def interactive_mode(self):
        """Interactive mode for manual testing"""
        logger.info("Entering interactive mode. Type 'quit' to exit.")
        
        try:
            while self.connected:
                try:
                    user_input = input("Enter message to send (or 'quit'): ").strip()
                    if user_input.lower() in ['quit', 'exit', 'q']:
                        break
                    
                    if user_input:
                        self.send_message(user_input)
                    
                except KeyboardInterrupt:
                    logger.info("Keyboard interrupt in interactive mode")
                    break
                except EOFError:
                    logger.info("EOF received, exiting interactive mode")
                    break
                    
        except Exception as e:
            logger.error(f"Error in interactive mode: {e}")

def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info(f"Received signal {signum}, shutting down...")
    if 'client' in globals():
        client.disconnect()
    sys.exit(0)

def run_performance_test(client, duration=30):
    """Run a performance test"""
    logger.info(f"Starting performance test for {duration} seconds")
    
    start_time = time.time()
    messages_sent = 0
    errors = 0
    
    while time.time() - start_time < duration:
        message = f"Performance test message {messages_sent + 1}"
        if client.send_message(message):
            messages_sent += 1
        else:
            errors += 1
            break
        
        # Small delay to avoid overwhelming
        time.sleep(0.1)
    
    actual_duration = time.time() - start_time
    logger.info(f"Performance test completed:")
    logger.info(f"  - Duration: {actual_duration:.2f} seconds")
    logger.info(f"  - Messages sent: {messages_sent}")
    logger.info(f"  - Errors: {errors}")
    logger.info(f"  - Rate: {messages_sent/actual_duration:.2f} messages/second")

if __name__ == "__main__":
    # Set up signal handlers for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Parse command line arguments for different test modes
    import argparse
    
    parser = argparse.ArgumentParser(description='TCP Client for FRP testing')
    parser.add_argument('--host', default='10.26.116.54', help='FRP server host')
    parser.add_argument('--port', type=int, default=35560, help='FRP server port')
    parser.add_argument('--mode', choices=['auto', 'interactive', 'performance'], 
                       default='auto', help='Test mode')
    parser.add_argument('--count', type=int, default=5, help='Number of test messages (auto mode)')
    parser.add_argument('--interval', type=float, default=2.0, help='Interval between messages (auto mode)')
    parser.add_argument('--duration', type=int, default=30, help='Duration for performance test (seconds)')
    
    args = parser.parse_args()
    
    # Create and connect client
    client = TCPClient(args.host, args.port)
    
    try:
        logger.info(f"Starting TCP test client for FRP testing (mode: {args.mode})")
        
        if not client.connect():
            logger.error("Failed to connect, exiting")
            sys.exit(1)
        
        if args.mode == 'auto':
            client.send_test_messages(args.count, args.interval)
        elif args.mode == 'interactive':
            client.interactive_mode()
        elif args.mode == 'performance':
            run_performance_test(client, args.duration)
            
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
    finally:
        client.disconnect()
        logger.info("TCP client stopped")
