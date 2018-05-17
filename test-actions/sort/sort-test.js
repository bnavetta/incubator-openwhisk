const fs = require('fs');
const os = require("os");
const path = require("path");

const chalk = require('chalk');
const dotenv = require("dotenv");
const openwhisk = require('openwhisk');
const request = require('request');

const REGISTRY_ADDR = '10.0.87.4';
// const REGISTRY_ADDR = '52.90.40.79';
const REGISTRY_PORT = 7777;

// const SORT_SERVER_ADDR = '10.0.87.5';
const EXTERNAL_SORT_SERVER_HOST = '52.90.40.79';
const SORT_SERVER_ADDR = EXTERNAL_SORT_SERVER_HOST;
const SORT_SERVER_PORT = 4343;

dotenv.load({
    path: path.join(os.homedir(), '.wskprops')
});

const ow = openwhisk({
    apihost: process.env['APIHOST'],
    api_key: process.env['AUTH'],
    ignore_certs: true,
});

// https://stackoverflow.com/questions/951021/what-is-the-javascript-version-of-sleep
function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function awaitResult(activationId) {
    try {
        return await ow.activations.result(activationId);
    } catch (err) {
        if (err.statusCode == 404) {
            await sleep(1000);
            return awaitResult(activationId);
        } else {
            throw err;
        }
    }
}

function launchAssigner(partitionId) {
    console.log('Launching assigner ' + chalk.blue(partitionId));
    return ow.actions.invoke({
        name: 'assignerGo',
        params: {
            registry_host: REGISTRY_ADDR,
            registry_port: REGISTRY_PORT,
            file_host: SORT_SERVER_ADDR,
            file_port: SORT_SERVER_PORT,
            id: partitionId,
        }
    }).then(({ activationId }) => {
        console.log('Assigner ' + chalk.blue(partitionId) + ' activation: ' + chalk.green(activationId));
        return awaitResult(activationId);
    });
}

function launchSorter(key, partitions) {
    console.log('Launching sorter for key ' + chalk.red(key));
    return ow.actions.invoke({
        name: 'sorter',
        result: true,
        params: {
            registry_host: REGISTRY_ADDR,
            registry_port: REGISTRY_PORT,
            file_host: SORT_SERVER_ADDR,
            file_port: SORT_SERVER_PORT,
            name: key,
            assigners: partitions,
        }
    }).then(({ activationId }) => {
        console.log('Sorter ' + chalk.red(key) + ' activation: ' + chalk.green(activationId));

        return awaitResult(activationId);
    });
}

function launchAll(partitions) {
    // const sortKeys = 'abcdefghijklmnopqrstuvwxyz'.split('');
    const sortKeys = ['a', 'b'];
    const sorters = sortKeys.map(key => launchSorter(key, partitions));

    const assigners = [];
    for (let i = 0; i < partitions; i++) {
        assigners.push(launchAssigner(i));
    }

    const sortPromise = Promise.all(sorters).then(sortResults => {
        console.log('Results from sorters:');
        for (let res of sortResults) {
            console.log(res);
        }
    });



    const assignPromise = sleep(2000).then(() => Promise.all(assigners).then(assignResults => {
        console.log('Results from assigners:');
        for (let res of assignResults) {
            console.log(res);
        }
    }));

    return Promise.all([sortPromise, assignPromise]).catch(e => {
        console.error(chalk.red('Fatal error!'));
        console.error(e);
    })
}

function createPartitions(dataset, partitions, sourceFile) {
    return new Promise((resolve, reject) => {
        console.log('Creating ' + chalk.green(partitions) + ' input partitions from ' + chalk.green(sourceFile));
        const sourceStream = fs.createReadStream(sourceFile);
        const url = `http://${EXTERNAL_SORT_SERVER_HOST}:${SORT_SERVER_PORT}/${dataset}?partitions=${partitions}`;
        const req = request
            .put(url, (error, response, body) => {
                if (error) {
                    reject(error);
                } else {
                    if (response.statusCode == 200) {
                        resolve(body);
                    } else {
                        reject(new Error(body));
                    }
                }
            });
        sourceStream.pipe(req);
    });
}

const partitions = 2;

createPartitions('sort', partitions, 'wordlist')
    .then(response => {
        console.log(chalk.green('Created input partitions'));
        console.log(response);
    })
    .catch(error => {
        console.error(chalk.red('Failed to create partitions'));
        console.error(error);
        process.exit(1);
    }).then(() => launchAll(partitions));