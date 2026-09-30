#!/usr/bin/env python3
"""
seed_demo_data.py: Populates Redis with realistic, multi-type sample data
for demonstrating the interactive FZF explorer and live preview capabilities.
Supports String, JSON, Hash, List, Set, Sorted Set (ZSET), Stream, and Binary.
"""

import json
import socket
import sys
import time

def resp_command(sock, *args):
    buf = [f"*{len(args)}\r\n".encode("utf-8")]
    for a in args:
        if isinstance(a, str):
            b = a.encode("utf-8")
        elif isinstance(a, bytes):
            b = a
        else:
            b = str(a).encode("utf-8")
        buf.append(f"${len(b)}\r\n".encode("utf-8"))
        buf.append(b)
        buf.append(b"\r\n")
    sock.sendall(b"".join(buf))
    return sock.recv(4096)

def seed(host="127.0.0.1", port=6379):
    print(f"Connecting to Redis at {host}:{port}...")
    try:
        s = socket.create_connection((host, port), timeout=3)
    except Exception as e:
        print(f"Error connecting to Redis: {e}")
        sys.exit(1)

    # Verify PING
    res = resp_command(s, "PING")
    if not res.startswith(b"+PONG"):
        print(f"Unexpected ping response: {res}")
        sys.exit(1)
    print("✓ Connected to Redis (PONG)")

    # 1. JSON Data (Stored as String with valid JSON content)
    user_profile = {
        "id": 1001,
        "username": "alex.morgan",
        "email": "alex.morgan@capstone.dev",
        "full_name": "Alex Morgan",
        "roles": ["developer", "admin", "billing_admin"],
        "status": "active",
        "profile": {
            "avatar_url": "https://avatar.dev/alex",
            "theme": "dark",
            "locale": "en-US",
            "timezone": "Asia/Ho_Chi_Minh",
            "notifications": {
                "email": True,
                "push": True,
                "slack": False
            }
        },
        "metadata": {
            "login_count": 142,
            "last_ip": "118.69.182.45",
            "created_at": "2026-01-15T08:30:00Z"
        }
    }
    resp_command(s, "SET", "user:profile:1001", json.dumps(user_profile, indent=2))
    resp_command(s, "EXPIRE", "user:profile:1001", 86400) # 24h TTL

    order_checkout = {
        "order_id": "ord-98234",
        "customer_id": 1001,
        "currency": "USD",
        "items": [
            {"sku": "LAPTOP-M4-PRO", "title": "MacBook Pro M4 36GB", "price": 2499.00, "qty": 1},
            {"sku": "DOCK-TB5-4K", "title": "Thunderbolt 5 Triple Display Dock", "price": 299.00, "qty": 1},
            {"sku": "CABLE-TB5-2M", "title": "Braided 2m TB5 Cable", "price": 49.00, "qty": 2}
        ],
        "subtotal": 2896.00,
        "tax": 231.68,
        "total": 3127.68,
        "shipping_address": {
            "recipient": "Alex Morgan",
            "address_line1": "72 Le Thanh Ton",
            "district": "District 1",
            "city": "Ho Chi Minh City",
            "country": "Vietnam"
        },
        "payment": {
            "method": "sepay_vietqr",
            "status": "COMPLETED",
            "reference": "SEPAY-TXN-8839210"
        },
        "created_at": "2026-09-29T14:45:10Z"
    }
    resp_command(s, "SET", "order:checkout:ord-98234", json.dumps(order_checkout, indent=2))

    features_config = {
        "maintenance_mode": False,
        "beta_ai_chat": True,
        "max_upload_size_mb": 100,
        "supported_locales": ["en-US", "vi-VN", "ja-JP"],
        "rate_limits": {
            "anonymous_per_min": 60,
            "authenticated_per_min": 600,
            "admin_per_min": 3000
        },
        "endpoints": {
            "api_gateway": "https://api.capstone.dev/v1",
            "websocket": "wss://ws.capstone.dev/realtime",
            "storage": "https://storage.capstone.dev/s3"
        }
    }
    resp_command(s, "SET", "system:config:features", json.dumps(features_config, indent=2))

    session_analytics = {
        "session_id": "sess-9a7c3b",
        "ip": "118.69.182.45",
        "browser": "Chrome 128.0.0",
        "os": "Linux x86_64",
        "screen_resolution": "2560x1440",
        "referrer": "https://google.com/search?q=jameskit",
        "page_views": 8,
        "events": [
            {"event": "page_view", "url": "/", "ts": 1727620000},
            {"event": "click", "element": "btn_get_started", "ts": 1727620015},
            {"event": "search", "query": "redis cli fzf", "ts": 1727620040}
        ]
    }
    resp_command(s, "SET", "analytics:session:sess-9a7c3b", json.dumps(session_analytics, indent=2))
    resp_command(s, "EXPIRE", "analytics:session:sess-9a7c3b", 7200) # 2h TTL

    # 2. Plain Strings
    resp_command(s, "SET", "auth:token:jwt-user-1001", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMDAxIiwibmFtZSI6IkFsZXggTW9yZ2FuIiwiaWF0IjoxNzI3NjIwMDAwLCJleHAiOjE3Mjc2MjM2MDB9.sample_signature_hash")
    resp_command(s, "EXPIRE", "auth:token:jwt-user-1001", 3600) # 1h TTL

    resp_command(s, "SET", "cache:api:exchange_rates", "USD_VND=25450.0;EUR_VND=27800.5;JPY_VND=175.2;GBP_VND=33100.0")
    resp_command(s, "EXPIRE", "cache:api:exchange_rates", 300) # 5m TTL

    resp_command(s, "SET", "maintenance:banner:system", "Notice: Scheduled database maintenance window on Sunday 02:00 AM UTC.")
    resp_command(s, "SET", "counter:hits:homepage", "149821")

    # 3. Hashes
    resp_command(s, "HSET", "user:session:h-user-1001",
                 "user_id", "1001",
                 "username", "alex.morgan",
                 "role", "admin",
                 "device", "Desktop Chrome / Linux",
                 "ip_address", "118.69.182.45",
                 "last_active", "2026-09-29 22:10:00",
                 "mfa_verified", "true",
                 "auth_provider", "better-auth")
    resp_command(s, "EXPIRE", "user:session:h-user-1001", 1800) # 30m TTL

    resp_command(s, "HSET", "product:inventory:prod-550",
                 "sku", "LAPTOP-M4-PRO",
                 "title", "MacBook Pro M4 36GB / 1TB Space Black",
                 "price", "2499.00",
                 "currency", "USD",
                 "stock_quantity", "42",
                 "reserved", "3",
                 "warehouse_zone", "WH-SEA-01",
                 "supplier", "Apple Distribution Inc.")

    resp_command(s, "HSET", "metrics:http:status_codes",
                 "200_ok", "89420",
                 "201_created", "4120",
                 "204_no_content", "1850",
                 "400_bad_request", "245",
                 "401_unauthorized", "89",
                 "403_forbidden", "14",
                 "404_not_found", "520",
                 "500_internal_error", "7")

    # 4. Lists
    resp_command(s, "DEL", "queue:jobs:email_notifications")
    resp_command(s, "RPUSH", "queue:jobs:email_notifications",
                 json.dumps({"job_id": "job-101", "type": "welcome_email", "to": "alex@dev.io", "retry": 0}),
                 json.dumps({"job_id": "job-102", "type": "order_receipt", "to": "sara@corp.net", "retry": 0}),
                 json.dumps({"job_id": "job-103", "type": "password_reset", "to": "kevin@cloud.co", "retry": 1}),
                 json.dumps({"job_id": "job-104", "type": "invoice_pdf", "to": "billing@acme.org", "retry": 0}))

    resp_command(s, "DEL", "feed:activity:user:1001")
    resp_command(s, "RPUSH", "feed:activity:user:1001",
                 "2026-09-29 22:00 - User alex.morgan created new context 'capstone-redis-common'",
                 "2026-09-29 21:45 - User alex.morgan executed 'redis explore' with FZF preview",
                 "2026-09-29 21:30 - User alex.morgan inspected key 'system:config:features'",
                 "2026-09-29 20:15 - User alex.morgan switched context to 'capstone-redis-001'")

    # 5. Sets
    resp_command(s, "DEL", "acl:roles:admin:permissions")
    resp_command(s, "SADD", "acl:roles:admin:permissions",
                 "users:create", "users:read", "users:update", "users:delete",
                 "system:config", "billing:manage", "audit:logs")

    resp_command(s, "DEL", "tags:popular:articles")
    resp_command(s, "SADD", "tags:popular:articles",
                 "golang", "redis", "microservices", "fzf-tui", "docker", "kubernetes", "event-driven")

    resp_command(s, "DEL", "security:blacklist:ips")
    resp_command(s, "SADD", "security:blacklist:ips",
                 "198.51.100.22", "203.0.113.88", "192.0.2.145", "198.51.100.199")

    # 6. Sorted Sets (ZSET)
    resp_command(s, "DEL", "leaderboard:gamification:weekly")
    resp_command(s, "ZADD", "leaderboard:gamification:weekly",
                 "14250.0", "player:hyperion",
                 "12800.5", "player:valkyrie",
                 "11950.0", "player:phoenix",
                 "9420.25", "player:cyber_ninja",
                 "8650.0", "player:shadow_walker",
                 "7200.0", "player:iron_clad")

    resp_command(s, "DEL", "ratelimit:sliding:api_client_789")
    now_ts = int(time.time())
    resp_command(s, "ZADD", "ratelimit:sliding:api_client_789",
                 str(now_ts - 50), "req-uuid-001",
                 str(now_ts - 30), "req-uuid-002",
                 str(now_ts - 15), "req-uuid-003",
                 str(now_ts - 2),  "req-uuid-004")
    resp_command(s, "EXPIRE", "ratelimit:sliding:api_client_789", 60)

    # 7. Streams
    resp_command(s, "XADD", "stream:events:audit_log", "*",
                 "action", "login",
                 "actor", "alex.morgan",
                 "ip", "118.69.182.45",
                 "status", "success")
    resp_command(s, "XADD", "stream:events:audit_log", "*",
                 "action", "context_create",
                 "actor", "alex.morgan",
                 "name", "capstone-redis-realtime",
                 "endpoint", "127.0.0.1:6380")
    resp_command(s, "XADD", "stream:events:audit_log", "*",
                 "action", "schema_sync",
                 "actor", "system",
                 "version", "v2.4.0",
                 "duration_ms", "120")

    # 8. Binary Data
    raw_binary = b"\x08\x96\x01\x12\x16ProtobufMessagePayload\x00\x01\x02\xff\xfe\x00\x10\x20\x30\x40\x00\x00\xaa\xbb\xcc"
    resp_command(s, "SET", "cache:binary:protobuf_sample", raw_binary)
    resp_command(s, "EXPIRE", "cache:binary:protobuf_sample", 86400)

    s.close()
    print("✓ Successfully populated demo data across String, JSON, Hash, List, Set, ZSET, Stream, and Binary!")

if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 6379
    seed(port=port)
