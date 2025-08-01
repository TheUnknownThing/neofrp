from flask import Blueprint, render_template, redirect, url_for, flash, request
from flask_login import login_user, logout_user, login_required, current_user
from app import db
from models import User, Tunnel
import secrets
from functools import wraps

bp = Blueprint('routes', __name__)

def admin_required(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        if not current_user.is_admin:
            flash("You do not have permission to access this page.")
            return redirect(url_for('routes.index'))
        return f(*args, **kwargs)
    return decorated_function

@bp.route('/', methods=['GET', 'POST'])
@login_required
def index():
    if request.method == 'POST':
        protocol = request.form['protocol']
        try:
            local_port = int(request.form['local_port'])
        except ValueError:
            flash("Local port must be a number.")
            return redirect(url_for('routes.index'))

        # Port allocation logic
        config = get_server_config()
        port_key = f"{protocol}_ports"
        available_ports = config.get('connections', {}).get(port_key, [])

        if not available_ports:
            flash(f"No {protocol.upper()} ports are configured on the server.")
            return redirect(url_for('routes.index'))

        assigned_ports = [t.server_port for t in Tunnel.query.all()]

        server_port = None
        for port in available_ports:
            if port not in assigned_ports:
                server_port = port
                break

        if server_port is None:
            flash("No available server ports for the selected protocol.")
            return redirect(url_for('routes.index'))

        new_tunnel = Tunnel(
            owner=current_user,
            protocol=protocol,
            local_port=local_port,
            server_port=server_port
        )
        db.session.add(new_tunnel)
        db.session.commit()
        flash(f"Tunnel created successfully! {current_user.token}:{local_port} -> your.server.ip:{server_port}")
        return redirect(url_for('routes.index'))

    tunnels = current_user.tunnels
    return render_template('index.html', tunnels=tunnels)

@bp.route('/delete_tunnel/<int:tunnel_id>', methods=['POST'])
@login_required
def delete_tunnel(tunnel_id):
    tunnel = Tunnel.query.get_or_404(tunnel_id)
    if tunnel.owner != current_user:
        flash("You do not have permission to delete this tunnel.")
        return redirect(url_for('routes.index'))

    db.session.delete(tunnel)
    db.session.commit()
    flash("Tunnel deleted successfully.")
    return redirect(url_for('routes.index'))

@bp.route('/login', methods=['GET', 'POST'])
def login():
    if current_user.is_authenticated:
        return redirect(url_for('routes.index'))
    if request.method == 'POST':
        username = request.form['username']
        password = request.form['password']
        user = User.query.filter_by(username=username).first()
        if user and user.check_password(password):
            login_user(user)
            return redirect(url_for('routes.index'))
        else:
            flash('Invalid username or password')
    return render_template('login.html')

@bp.route('/register', methods=['GET', 'POST'])
def register():
    if current_user.is_authenticated:
        return redirect(url_for('routes.index'))
    if request.method == 'POST':
        username = request.form['username']
        password = request.form['password']
        if User.query.filter_by(username=username).first():
            flash('Username already exists')
        else:
            new_user = User(username=username, token=secrets.token_hex(16))
            new_user.set_password(password)
            # The first registered user is an admin
            if User.query.count() == 0:
                new_user.is_admin = True
            db.session.add(new_user)
            db.session.commit()
            flash('Registration successful, please login.')
            return redirect(url_for('routes.login'))
    return render_template('register.html')

import json
import os

SERVER_CONFIG_PATH = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'server.json')

def get_server_config():
    try:
        with open(SERVER_CONFIG_PATH, 'r') as f:
            return json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        # File doesn't exist or is invalid, create a default one
        default_config = {
            "log": {
                "log_level": "info"
            },
            "recognized_tokens": [],
            "transport": {
                "protocol": "quic",
                "port": 3400
            },
            "connections": {
                "tcp_ports": list(range(35000, 35100)),
                "udp_ports": list(range(36000, 36100))
            }
        }
        save_server_config(default_config)
        return default_config

def save_server_config(config):
    with open(SERVER_CONFIG_PATH, 'w') as f:
        json.dump(config, f, indent=4)

@bp.route('/logout')
@login_required
def logout():
    logout_user()
    return redirect(url_for('routes.login'))

@bp.route('/admin', methods=['GET', 'POST'])
@login_required
@admin_required
def admin():
    # This POST handling is for the 'Create User' form
    if request.method == 'POST' and 'username' in request.form:
        username = request.form['username']
        password = request.form['password']
        if User.query.filter_by(username=username).first():
            flash('Username already exists')
        else:
            new_user = User(username=username, token=secrets.token_hex(16))
            new_user.set_password(password)
            db.session.add(new_user)

            # Update server.json
            config = get_server_config()
            if 'recognized_tokens' not in config:
                config['recognized_tokens'] = []
            config['recognized_tokens'].append(new_user.token)
            save_server_config(config)

            db.session.commit()
            flash('User created successfully.')
        return redirect(url_for('routes.admin'))

    users = User.query.order_by(User.id).all()
    server_running = is_frps_running()
    return render_template('admin.html', users=users, server_running=server_running)

import subprocess
from flask import jsonify

PID_FILE_PATH = os.path.join(os.path.dirname(__file__), 'instance', 'frps.pid')
FRPS_BINARY_PATH = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'bin', 'frps')

def is_frps_running():
    if not os.path.exists(PID_FILE_PATH):
        return False
    with open(PID_FILE_PATH, 'r') as f:
        pid = f.read().strip()
        if not pid:
            return False
        try:
            os.kill(int(pid), 0)
        except (OSError, ValueError):
            return False
        else:
            return True

@bp.route('/admin/delete_user/<int:user_id>', methods=['POST'])
@login_required
@admin_required
def delete_user(user_id):
    user_to_delete = User.query.get_or_404(user_id)
    if user_to_delete.is_admin:
        flash("You cannot delete an admin account.")
        return redirect(url_for('routes.admin'))

    # Update server.json
    config = get_server_config()
    if 'recognized_tokens' in config:
        config['recognized_tokens'] = [t for t in config['recognized_tokens'] if t != user_to_delete.token]
        save_server_config(config)

    db.session.delete(user_to_delete)
    db.session.commit()
    flash('User deleted successfully.')
    return redirect(url_for('routes.admin'))

@bp.route('/admin/start_server', methods=['POST'])
@login_required
@admin_required
def start_server():
    if is_frps_running():
        flash("Server is already running.")
        return redirect(url_for('routes.admin'))

    # Ensure server.json exists and is valid
    get_server_config()

    process = subprocess.Popen([FRPS_BINARY_PATH, '-c', SERVER_CONFIG_PATH], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    with open(PID_FILE_PATH, 'w') as f:
        f.write(str(process.pid))

    flash("neofrp server started.")
    return redirect(url_for('routes.admin'))

@bp.route('/admin/stop_server', methods=['POST'])
@login_required
@admin_required
def stop_server():
    if not is_frps_running():
        flash("Server is not running.")
        return redirect(url_for('routes.admin'))

    with open(PID_FILE_PATH, 'r') as f:
        pid = int(f.read().strip())

    try:
        os.kill(pid, subprocess.SIGTERM)
    except OSError:
        flash("Could not stop server. Process not found.")

    if os.path.exists(PID_FILE_PATH):
        os.remove(PID_FILE_PATH)

    flash("neofrp server stopped.")
    return redirect(url_for('routes.admin'))

@bp.route('/generate_config')
@login_required
def generate_config():
    user = current_user
    server_config = get_server_config()

    client_connections = []
    for tunnel in user.tunnels:
        client_connections.append({
            "type": tunnel.protocol,
            "local_port": tunnel.local_port,
            "server_port": tunnel.server_port
        })

    client_config = {
        "log": {
            "log_level": "info"
        },
        "token": user.token,
        "transport": {
            "protocol": server_config.get("transport", {}).get("protocol", "quic"),
            "server_ip": "your.server.ip",
            "server_port": server_config.get("transport", {}).get("port", 3400)
        },
        "connections": client_connections
    }

    response = jsonify(client_config)
    response.headers['Content-Disposition'] = f'attachment; filename={user.username}_client.json'
    return response
