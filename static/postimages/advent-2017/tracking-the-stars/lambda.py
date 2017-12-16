# This uses the Python 2.7 lambda runtime
'''Lambda function to retrieve GitHub stars and write them to DynamoDB'''

from __future__ import print_function

import json
from datetime import datetime
from time import time
from urllib2 import urlopen

import boto3

SEARCH_URL_BASE = "https://api.github.com/search/repositories?q=language:Go&sort=stars&per_page=100&page="


def page_url(page):
    '''Generates the search URL for a given page number'''
    return SEARCH_URL_BASE + str(page)


def lambda_handler(event, context):
    '''Main lambda handler'''

    now = int(time())

    try:
        # Connect to DynamoDB first
        # In case there are any issues, we won't waste time reading the stars
        dynamodb = boto3.resource(
            'dynamodb',
            aws_access_key_id='HAHAHAHAHAHAHAHAHAHA',
            aws_secret_access_key='LOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLO'
        )

        table = dynamodb.Table('github-stars')
        table.load()

        # First, retrieve all the GitHub stars information
        print("Retrieving search results")
        ret = {}

        for i in xrange(1, 11):
            print("Retrieving page {} of search results".format(i))
            data = json.load(urlopen(page_url(i)))

            for item in data['items']:
                ret[item['full_name']] = item['stargazers_count']

        print(len(ret), "repos retrieved. Writing to DynamoDB.")

        # Now write to DynamoDB

        with table.batch_writer() as batch:
            for repo, stars in ret.iteritems():
                batch.put_item(Item={
                    'repo':  repo,
                    'stars': stars,
                    'ts':    now,
                })

    except:
        print('FAILURE!')
        raise
    else:
        print('SUCCESS!')
    finally:
        print('Complete at {}'.format(str(datetime.now())))


if __name__ == "__main__":
    lambda_handler(None, None)
