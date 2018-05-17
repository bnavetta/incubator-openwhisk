const os = require("os");
const path = require("path");

const dotenv = require("dotenv");
const openwhisk = require('openwhisk');

const REGISTRY_ADDR = '10.0.87.5:7777';

dotenv.load({
    path: path.join(os.homedir(), '.wskprops')
});

const ow = openwhisk({
    apihost: process.env['APIHOST'],
    api_key: process.env['AUTH'],
    ignore_certs: true,
});

function spawnLambda(id, number, instances) {
    return ow.actions.invoke({
        name: 'allToAll',
        result: true, blocking: true,
        params: { registry: REGISTRY_ADDR, id, myNumber: number, instances }
    }).then(result => {
        console.log(`Received ${JSON.stringify(result)} from ${id}`);

        if (result.sum) {
            return result.sum;
        } else {
            return Promise.reject(new Error(result.error || 'Unknown error'));
        }
    });
}

function spawnAll(count) {
    let activations = [];
    for (let i = 0; i < count; i++) {
        // const num = Math.floor(Math.random() * 100);
        const num = i + 1;
        console.log(`Spawning lambda ${i} with number ${num}`);
        activations.push(spawnLambda(i, num, count));
    }

    return Promise.all(activations);
}

function verify(results) {
    if (results.length == 0) return true;

    const expected = results[0];
    for (let result of results) {
        if (result != expected) {
            return false;
        }
    }

    return true;
}

function run() {
    return spawnAll(5)
        .then(verify)
        .then(valid => console.log("All sums match? " + valid))
        .catch(e => console.error(e));
}

run();
