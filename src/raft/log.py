import re
import sys

import chardet

def detect_encoding(filename):
    with open(filename, 'rb') as f:
        result = chardet.detect(f.read())
    return result['encoding']


Servers = set()


def handle_vote(line, term_number):
    #016547  [Vote-2] p1(F,2): Vote Start
    #016547  [Vote-2] p1(F,2)->p4 "Ask Vote"
    #018249  [Vote-2] p0(F,2)->p1 "OKKKK"
    #004372  [Vote-1] p4(F,1): Got 3 votes
    pattern1 = re.compile(r'\bp(\d+)')
    pattern2 = re.compile(r'\bp(\d+).*?->p(\d+)')

    match1 = pattern1.search(line)
    match2 = pattern2.search(line)

    if match1 :
        n1 = match1.group(1)
        if n1 in Servers:
            print(line)
    elif match2:
        n1 = match2.group(1)
        n2 = match2.group(2)
        if n1 in Servers and n2 in Servers:
            print(line)
    return True

def handle_start(line):
    pattern = re.compile(r'\bp(\d+)')
    match = pattern.search(line)
    n1 = match.group(1)
    if n1 in Servers:
        print(line)
    return True

def handle_cmit(line):
    # [CMIT] p3(F,1) "..." 匹配 "3"
    pattern = re.compile(r'\bp(\d+)')
    match = pattern.search(line)
    n1 = match.group(1)
    if n1 in Servers:
        print(line)
    return True

def handle_aply(line):
    # [APLY] p3(F,1) "..." 匹配 "3"
    pattern = re.compile(r'\bp(\d+)')
    match = pattern.search(line)
    n1 = match.group(1)
    if n1 in Servers:
        print(line)
    return True

def handle_info(line):
    print(line)
    return True

def handle_aped(line):
    #[APED] p4(L,1)->p0 ".." 匹配"4"和"0" 
    #[APED] p3(F,1) ".." 匹配"3"
    pattern1 = re.compile(r'\bp(\d+)')
    pattern2 = re.compile(r'\bp(\d+).*?->p(\d+)')

    match1 = pattern1.search(line)
    match2 = pattern2.search(line)

    if match1 and not match2:
        n1 = match1.group(1)
        if n1 in Servers:
            print(line)
    elif match2:
        n1 = match2.group(1)
        n2 = match2.group(2)
        if n1 in Servers and n2 in Servers:
            print(line)
    return True

def handle_other(line):
    return False

def check_connect(line):
    connect_re = re.compile(r'^connect\((\d+)\)$')
    match = connect_re.search(line.strip())
    if match:
        Servers.add(match.group(1))
        print(Servers)
        return True
    else:
        return False
    
def check_disconnect(line):
    connect_re = re.compile(r'^disconnect\((\d+)\)$')
    match = connect_re.search(line)
    if match:
        Servers.discard(match.group(1))
        if not Servers:
            return True
        print(Servers)
        return True
    else:
        return False

def check_message(line):
    type_re = re.compile(r'\[(.*?)(?:-(\d+))?\]')
    match = type_re.search(line)
    if match:
        log_type = match.group(1)
        term_number = match.group(2)
        if log_type == 'Info' and handle_info(line):
            return True
        elif log_type == 'APED' and handle_aped(line):
            return True
        elif log_type == 'CMIT' and handle_cmit(line):
            return True
        elif log_type == 'APLY' and handle_aply(line):
            return True
        elif log_type == 'VOTE' and handle_vote(line, term_number):
            return True
        elif log_type == 'Start' and handle_start(line):
            return True
        elif log_type == 'HERT':
            return True
    else:
        return False


def filter_logs(filename):
    encoding = detect_encoding(filename)
    with open(filename, 'r', encoding=encoding) as f:
        lines = f.readlines()

    for line in lines:
        if check_connect(line):
            continue
        if check_disconnect(line):
            continue
        if check_message(line):
            continue
        print(line)


if len(sys.argv) < 2:
    print("Usage: python log.py <filename>")
    sys.exit(1)

filename = sys.argv[1]
filter_logs(filename)