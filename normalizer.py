#!/usr/bin/env python3

# ---------------------------------------------------------------------
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
# ----------------------------------------------------------------------

import os
import re
import sys
import time
import json
import argparse

# EXAMPLE FORMAT:  {"email": "jdoe@example.com", "domain": "example.com", "password": "monkey123", "user": "jdoe"}

INFO = "\033[1m\033[36m[*]\033[0m "
WARN = "\033[1m\033[31m[!]\033[0m "


def display_info(msg):
    ''' Clearline and print message '''
    sys.stdout.write(chr(27) + '[2K')
    sys.stdout.write('\r' + INFO + msg)
    sys.stdout.flush()


#
# >>> PARSERS
#
class Parser(object):
    '''
    Main parser class, implements basic functionality to scan a file system
    and save the output in json/newline delimited format.
    '''

    name = 'base'
    email_regex = r"(^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]{2,63})"
    password_regex = r"[^\x00-\x1F\x80-\x9F]*"

    def __init__(self, output, append=True, skip_prefix='', skip_suffix=''):
        self._skip_prefix = skip_prefix
        self._skip_suffix = skip_suffix
        mode = 'a' if append else 'w'
        display_info('Output (%s): %s\n' % (mode, output))
        self._output = open(output, mode)

    def parse(self, targets, recursive):
        ''' Parse all target(s) '''
        for target in targets:
            display_info('Target is: %s (dir: %s, r: %s)\n' %
                         (target, os.path.isdir(target), recursive))
            if os.path.isdir(target) and recursive:
                self.rscan_directory(target)
            elif os.path.isdir(target):
                self.scan_directory(target)
            else:
                self.scan_file(target)

    def rscan_directory(self, target):
        ''' Recursively scan directory '''
        display_info('Walking: %s ...\n' % target)
        for root, _, files in os.walk(target, topdown=False):
            for name in files:
                display_info('File: %s ...\n' % name)
                self.scan_file(os.path.join(root, name))

    def scan_directory(self, target):
        ''' Scan a directory '''
        ls = [os.path.join(target, name) for name in os.listdir(target)]
        ls = [entry for entry in ls if os.path.isfile(entry)]
        for entry in ls:
            self.scan_file(entry)

    def scan_file(self, target):
        ''' Scan a target file (or possibly skip depending on config) '''
        filename = os.path.basename(target)
        if self._skip_prefix and filename.startswith(self._skip_prefix):
            display_info('Skip %s\n' % filename)
            return
        if self._skip_suffix and filename.endswith(self._skip_suffix):
            display_info('Skip %s\n' % filename)
            return
        display_info('Scanning: %s\n' % target)
        with open(target, 'r', encoding='utf-8', errors='ignore') as fp:
            self.walk_file(fp)

    def walk_file(self, fp):
        ''' Walk a file line by line '''
        for count, line in enumerate(self.lazy_readline(fp)):
            display_info('Parsing line %d ...' % (count+1))
            if not len(line):
                continue
            try:
                values = self.parse_line(line)
                if values is not None:
                    self.save(*values)
            except Exception as error:
                print('\r'+WARN+str(error)+' (line number %s: %r)\n' %
                      (count+1, line))

    def lazy_readline(self, fp):
        ''' Lazily read line from file until EOF '''
        data = fp.readline()
        while data:
            yield data.strip()
            data = fp.readline()

    def save(self, email, user, domain, password):
        ''' Save results of parse_line()  '''
        self._output.write(json.dumps({
            'email': email,
            'user': user,
            'domain': domain,
            'password': password,
        }))
        self._output.write("\n")

    def parse_line(self, line):
        ''' Parse a single line, child class should implement this '''
        raise NotImplementedError()


class ColonNewlineParser(Parser):
    ''' Parses colon/newline delimited text files '''

    name = 'colon-newline'
    pattern = re.compile(Parser.email_regex+r':'+Parser.password_regex)

    def parse_line(self, line):
        ''' Parse a colon newline delimited line '''
        if not self.pattern.match(line):
            return None
        parts = line.split(':')
        email = parts[0]
        password = ''.join(parts[1:])
        if '@' in email:
            user, domain = email.split('@')
            return email, user, domain, password


class SemicolonNewlineParser(Parser):
    ''' Parses semicolon/newline delimited text files '''

    name = 'semicolon-newline'
    pattern = re.compile(Parser.email_regex+r';'+Parser.password_regex)

    def parse_line(self, line):
        ''' Parse a semicolon newline delimited line '''
        if not self.pattern.match(line):
            return None
        parts = line.split(';')
        email = parts[0]
        password = ''.join(parts[1:])
        if '@' in email:
            user, domain = email.split('@')
            return email, user, domain, password


class WhitespaceParser(Parser):
    ''' Parses whitespace seperated text files (e.g. MySQL OUTFILE's) '''

    name = 'whitespace'
    pattern = re.compile(Parser.email_regex+r'[ \t]+'+Parser.password_regex)

    def parse_line(self, line):
        ''' Parse a whitespace/newline delimited line '''
        if not self.pattern.match(line):
            return None
        parts = [part for part in line.split() if len(part)]
        email = parts[0]
        password = ''.join(parts[1:])
        if '@' in email:
            user, domain = email.split('@')
            return email, user, domain, password


#
# >>> MAIN
#
PARSERS = {
    ColonNewlineParser.name: ColonNewlineParser,
    SemicolonNewlineParser.name: SemicolonNewlineParser,
    WhitespaceParser.name: WhitespaceParser,
}


def main(args):
    ''' Validate all arguments look correct '''
    if args.format not in PARSERS:
        print(WARN + "Invalid parser '%s'", args.parser)
        return
    args.targets = [
        target for target in args.targets if os.path.exists(target)
    ]
    if 0 < len(args.targets):
        parser = PARSERS[args.format](
            args.output, args.append, args.skip_prefix, args.skip_suffix,
        )
        parser.parse(args.targets, args.recursive)
    else:
        print(WARN + "No valid targets found in %s" % args.targets)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Normalizes leaked database files for use in BigQuery')
    parser.add_argument('--target', '-t',
                        help='file or directory with leak file(s)',
                        dest='targets',
                        nargs='*',
                        required=True)
    parser.add_argument('--output', '-o',
                        dest='output',
                        help='output file with hooks (default: cwd)',
                        default='leaks.json')
    parser.add_argument('--format', '-f',
                        dest='format',
                        help='file format',
                        required=True,
                        choices=PARSERS.keys())
    parser.add_argument('--append', '-a',
                        help='append output file (default: false)',
                        action='store_true',
                        dest='append')
    parser.add_argument('--recursive', '-r',
                        help='recursively scan directories (default: false)',
                        action='store_true',
                        dest='recursive')
    parser.add_argument('--skip-suffix', '-s',
                        dest='skip_suffix',
                        default='',
                        help='skip files with a given suffix')
    parser.add_argument('--skip-prefix', '-S',
                        dest='skip_prefix',
                        default='',
                        help='skip files with a given prefix')
    try:
        main(parser.parse_args())
    except KeyboardInterrupt:
        display_info('User exit.')
