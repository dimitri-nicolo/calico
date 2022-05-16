from __future__ import print_function
import argparse
import glob
import json
import os.path
import sys

JOB_SUFFIX = "-job.json"
DATAFEED_SUFFIX = "-datafeed.json"


class ParseError(Exception):
    pass


def _get_blobs(data_dir, suffix):
    for job_filename in glob.glob(os.path.join(data_dir, "*"+suffix)):
        try:
            yield json.load(open(job_filename))
        except ValueError as e:
            raise ParseError('ParseError: file={} except={}'.format(
                job_filename, e))


def get_jobs(data_dir):
    return _get_blobs(data_dir, JOB_SUFFIX)


def get_datafeeds(data_dir):
    return _get_blobs(data_dir, DATAFEED_SUFFIX)


class ValidationError(Exception):
    pass


def validate(jobs, datafeeds):
    seen_jobs = set()
    for dfid, df in datafeeds.items():
        jid = df['job_id']
        if jid not in jobs:
            raise ValidationError(
                'Datafeed {} references undefined job {}'.format(
                    dfid, jid))
        seen_jobs.add(jid)

    if len(seen_jobs) != len(jobs):
        missed_jobs = set(jobs.keys()).difference(seen_jobs)

        raise ValidationError('No datafeeds defined for {}'.format(
            ','.join(sorted(missed_jobs))))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--data-dir', default=os.path.join(
        os.path.dirname(sys.argv[0]),
        "../install/data"))
    parser.add_argument('--outfile', type=argparse.FileType('w'),
                        default=sys.stdout)
    args = parser.parse_args()

    try:
        jobs = {}
        for job in get_jobs(args.data_dir):
            jobs[job['job_id']] = job

        datafeeds = {}
        for df in get_datafeeds(args.data_dir):
            datafeeds[df['datafeed_id']] = df
    except ParseError as e:
        print(e, file=sys.stderr)
        sys.exit(1)

    try:
        validate(jobs, datafeeds)
    except ValidationError as e:
        print(e, file=sys.stderr)
        sys.exit(1)

    json.dump({'jobs': jobs, 'datafeeds': datafeeds}, args.outfile, indent=4)
    args.outfile.write('\n')


if __name__ == "__main__":
    main()
