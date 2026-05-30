from procmon_report.normalize import normalize_key, redact_cmdline


def test_java_jar():
    cmd = "/usr/bin/java -Xmx8g -jar /opt/app/order-svc.jar --port=9090"
    assert normalize_key("java", cmd) == "java:order-svc"


def test_java_jar_uppercase_ext():
    cmd = "/usr/bin/java -jar /opt/Foo.JAR"
    assert normalize_key("java", cmd) == "java:Foo"


def test_java_main_class():
    cmd = "/usr/bin/java -Xmx4g com.example.UserService --port=8080"
    assert normalize_key("java", cmd) == "java:UserService"


def test_name_flag_space():
    cmd = "worker --name billing-cron --interval 30"
    assert normalize_key("worker", cmd) == "worker:billing-cron"


def test_name_flag_equals():
    cmd = "worker --name=billing-cron"
    assert normalize_key("worker", cmd) == "worker:billing-cron"


def test_plain_comm_fallback():
    assert normalize_key("mysqld", "mysqld --defaults-file=/etc/my.cnf") == "mysqld"


def test_empty_cmdline():
    # kernel threads typically have empty cmdline
    assert normalize_key("kthreadd", "") == "kthreadd"


def test_co_located_java_services_disambiguate():
    a = normalize_key("java", "java -jar order-svc.jar")
    b = normalize_key("java", "java -jar user-svc.jar")
    assert a != b
    assert a == "java:order-svc"
    assert b == "java:user-svc"


def test_redact_password_equals():
    cmd = "mysqld --user=root --password=hunter2 --port=3306"
    out = redact_cmdline(cmd)
    assert "hunter2" not in out
    assert "<redacted>" in out


def test_redact_token_space():
    cmd = "agent --token abc123 --host x"
    out = redact_cmdline(cmd)
    assert "abc123" not in out
