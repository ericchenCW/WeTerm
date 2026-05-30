"""Tests for the v2 normalize_key signature (cwd + container)."""
from procmon_report.normalize import normalize_key


def test_container_wins_over_java():
    # Container present → "<container>/<comm>" regardless of java rule.
    cmd = "/usr/bin/java -jar /opt/order-svc.jar"
    key = normalize_key("java", cmd, container="order-container")
    assert key == "order-container/java"


def test_container_wins_over_name_flag():
    key = normalize_key("celery", "celery --name billing", container="celery-worker")
    assert key == "celery-worker/celery"


def test_container_wins_over_cwd():
    key = normalize_key("python", "python app.py",
                        cwd="/srv/app-a", container="api-server")
    assert key == "api-server/python"


def test_cwd_disambiguates_python():
    a = normalize_key("python", "python app.py", cwd="/srv/app-a")
    b = normalize_key("python", "python app.py", cwd="/srv/app-b")
    assert a == "python@app-a"
    assert b == "python@app-b"


def test_cwd_default_root_no_suffix():
    assert normalize_key("bash", "bash -i", cwd="/") == "bash"
    assert normalize_key("bash", "bash -i", cwd="/root") == "bash"


def test_cwd_user_home_no_suffix():
    # /home/<user> is treated as default — too noisy to include
    assert normalize_key("bash", "bash", cwd="/home/eric") == "bash"
    # But a subdir under /home keeps signal (it's a project, not the home itself)
    assert normalize_key("bash", "bash",
                         cwd="/home/eric/myproj") == "bash@myproj"


def test_legacy_signature_still_works():
    # Calling with only (comm, cmdline) — no cwd/container — preserves
    # original behavior. Important for backward compat with old records.
    assert normalize_key("java", "java -jar foo.jar") == "java:foo"
    assert normalize_key("mysqld", "mysqld") == "mysqld"


def test_none_args_disable_rules():
    # Explicit None for cwd/container behaves the same as omitting them.
    assert normalize_key("java", "java -jar foo.jar",
                         cwd=None, container=None) == "java:foo"


def test_cwd_with_trailing_slash():
    assert normalize_key("python", "python",
                         cwd="/srv/app-a/") == "python@app-a"


def test_priority_order_full():
    # All four signals at once → container still wins.
    key = normalize_key(
        "java",
        "java -jar /opt/order.jar --name explicit",
        cwd="/srv/orders",
        container="orders-prod",
    )
    assert key == "orders-prod/java"


def test_priority_order_no_container():
    # Without container, java rule should win over --name and cwd.
    key = normalize_key(
        "java",
        "java -jar /opt/order.jar --name explicit",
        cwd="/srv/orders",
    )
    assert key == "java:order"


def test_priority_order_no_java_no_container():
    # No container, no java → --name beats cwd.
    key = normalize_key(
        "worker",
        "worker --name billing-cron",
        cwd="/srv/worker",
    )
    assert key == "worker:billing-cron"
