import re
import sys

Servers = set()

def check_message(line):
    return False
    type_re = re.compile(r'\[(.*?)(?:-(\d+))?\]')
    match = type_re.search(line)
    if match:
        log_type = match.group(1)
        term_number = match.group(2)
        if log_type == 'Info':
            handle_info(line)
        elif log_type == 'APED':
            handle_aped(line)
        elif log_type == 'CMIT':
            handle_cmit(line)
        elif log_type == 'APLY':
            handle_aply(line)
        elif log_type == 'Vote':
            handle_vote(line, term_number)
    else:
        print('No log type found')

def handle_vote(line):
    pass


def handle_cmit(line):
    # 在这里实现 Info 类型的日志处理
    pass

def handle_aply(line):
    # 在这里实现 Info 类型的日志处理
    pass

def handle_info(line):
    # 在这里实现 Info 类型的日志处理
    pass

def handle_aped(line):
    # 在这里实现 APED 类型的日志处理
    pass

def handle_other(line):
    # 在这里实现其他类型的日志处理
    pass



def check_connect(line):
    connect_re = re.compile(r'^connect\((\d+)\)$')
    match = connect_re.search(line.strip())
    if match:
        Servers.add(int(match.group(1)))
        print(Servers)
        return True
    else:
        return False
    
def check_disconnect(line):
    connect_re = re.compile(r'^disconnect\((\d+)\)$')
    match = connect_re.search(line)
    if match:
        Servers.discard(int(match.group(1)))
        if not Servers:
            return True
        print(Servers)
        return True
    else:
        return False

def check_message(line):
    message_re = re.compile(r'\b(p)(\d+)\(.*?\)->')
    match = message_re.search(line)
    if match:
        return int(match.group(2))
    else:
        return None

def filter_logs(filename):
    with open(filename, 'r') as f:
        lines = f.readlines()

    for line in lines:
        if check_connect(line):
            continue
        if check_disconnect(line):
            continue
        if check_message(line):
            continue
        #print(line)


if len(sys.argv) < 2:
    print("Usage: python log.py <filename>")
    sys.exit(1)

filename = sys.argv[1]
filter_logs(filename)