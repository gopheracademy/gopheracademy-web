#!/usr/local/bin/python3
'''Script to download a single repo's star history from DynamoDB'''

import sys
from datetime import datetime

import boto3
from boto3.dynamodb.conditions import Key

import matplotlib.pyplot as plt


def main(args):
    '''Main function'''

    if len(args) < 2:
        print('Must provide at least one repo name (e.g. golang/go)')
        sys.exit(1)

    # Connect to DynamoDB first
    dynamodb = boto3.resource(
        'dynamodb',
        aws_access_key_id='HAHAHAHAHAHAHAHAHAHAHA',
        aws_secret_access_key='LOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLOLO')

    table = dynamodb.Table('github-stars')
    table.load()

    plt.subplot(211)

    cache = {}

    args = args[1:]
    for repo in args:
        # Read all data for the requested repo
        response = table.query(KeyConditionExpression=Key('repo').eq(repo))
        items = response['Items']
        print(repo + ':', len(items), 'items')

        # Gather number of stars vs timestamp
        stars = []
        timestamps = []

        for item in items:
            stars.append(item['stars'])
            timestamps.append(datetime.fromtimestamp(item['ts']))

        plt.plot(timestamps, stars, label=repo)
        cache[repo] = (timestamps, stars)

    # plt.xlabel('Time')
    plt.xticks(rotation=25)

    plt.ylabel('# of stars')

    plt.grid(which='major')
    plt.grid(which='minor', linestyle='--')

    title = ''
    line = ''

    # Set up lines to be a max of 61 chars long (with the comma)
    # so it looks more natural
    for repo in args:
        if len(line) + len(repo) > 60:
            title += line + '\n'
            line = ''

        line += repo + ', '

    if len(line) > 0:
        title += line

    title = title.rstrip(',\n ')
    plt.title('Stars over time for\n' + title, loc='left')

    legend = plt.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)

    # now for the slope subplot
    plt.subplot(212)

    for repo in args:
        dat = cache[repo]
        timestamps = dat[0]
        stars = dat[1]

        slopes = []

        for i in range(len(stars)):
            if i == 0:
                continue

            slopes.append(stars[i] - stars[i - 1])

        # Remove the first timestamp so the data is showing the increase
        # ending at the time displayed
        timestamps = timestamps[1:]

        plt.plot(timestamps, slopes, label=repo)

    plt.xlabel('Time')
    plt.xticks(rotation=25)

    plt.ylabel('change in stars')

    plt.grid(which='major')
    plt.grid(which='minor', linestyle='--')

    legend = plt.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)

    plt.subplots_adjust(hspace=0.5)

    plt.savefig('fubar', bbox_extra_artists=(legend, ), bbox_inches='tight')
    plt.show()


if __name__ == '__main__':
    main(sys.argv)
