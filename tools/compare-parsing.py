# compare-parsing.py parses all programs in a CSV files with header location,code
# using a old and new runtime/cmd/parse program.
#
# It reports already broken programs, programs that are broken with the new parser,
# and when parses of the old and new parser differ

import csv
import json
import logging
import subprocess
import sys
import os
import prettydiff

def parse(path, prog):
    return json.loads(subprocess.getoutput(f"{prog} -json {path}"))[0]

logging.basicConfig(level=logging.INFO)

csv.field_size_limit(sys.maxsize)

[_, csv_path, directory, parse_old, parse_new] = sys.argv

with open(csv_path) as csv_file:
    csv_reader = csv.reader(csv_file)
    next(csv_reader)
    for row in csv_reader:
        [location, code] = row
        logging.info(location)
        contract_path = os.path.join(directory, f"{location}.cdc")
        with open(contract_path, 'w') as contract_file:
            contract_file.write(code)

        res1 = parse(contract_path, parse_old)
        if 'error' in res1:
            logging.warning(f"{location} is broken")
            print(res1['error']['Errors'])
            continue

        res2 = parse(contract_path, parse_new)
        if 'error' in res2:
            logging.error(f"{location} broke")
            print(res2['error']['Errors'])
            continue

        if res1 != res2:
            prettydiff.print_diff(res1, res2)
