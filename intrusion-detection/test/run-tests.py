import argparse
import datetime
import json
import os
import time

import elasticsearch as es


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--config', default='config.json')
    parser.add_argument('--elastic-host',
                        default=os.environ.get('ELASTIC_HOST', '127.0.0.1'))
    parser.add_argument('--elastic-port', type=int,
                        default=os.environ.get('ELASTIC_PORT', 9200))
    parser.add_argument('--elastic-use-ssl', type=bool,
                        default=os.environ.get('ELASTIC_SCHEME', 'http')
                        == 'https')
    args = parser.parse_args()

    config = json.load(open(args.config))

    c = es.Elasticsearch([{
        'host': args.elastic_host,
        'port': args.elastic_port,
        'use_ssl': args.elastic_use_ssl,
    }])
    ml = es.client.xpack.ml.MlClient(c)

    for dfid, df in config['datafeeds'].items():
        jid = df['job_id']

        print('Testing {}'.format(jid))

        # Check whether or not the job is open
        job_stats = ml.get_job_stats(job_id=jid)
        if job_stats['jobs'][0]['state'] == 'closed':
            # Open the job
            opened = ml.open_job(job_id=jid)['opened']
            if not opened:
                raise Exception('Did not open {}'.format(jid))

        # Check whether or not the datafeed is started
        datafeed_stats = ml.get_datafeed_stats(datafeed_id=dfid)
        if datafeed_stats['datafeeds'][0]['state'] == 'stopped':
            # Start the datafeed
            started = ml.start_datafeed(
                datafeed_id=dfid,
                end=datetime.datetime.utcnow()
            )['started']
            if not started:
                raise Exception('Did not start {}'.format(dfid))

        while True:
            datafeed_stats = ml.get_datafeed_stats(datafeed_id=dfid)
            if datafeed_stats['datafeeds'][0]['state'] == 'stopped':
                closed = ml.close_job(job_id=jid)['closed']
                if not closed:
                    raise Exception('Did not close {}'.format(jid))
                break
            else:
                time.sleep(1)


if __name__ == "__main__":
    main()
