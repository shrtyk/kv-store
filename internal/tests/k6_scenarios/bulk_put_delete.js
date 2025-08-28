import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL ? `${__ENV.BASE_URL}/v1` : 'http://localhost:16700/v1';
const keySize = 100;
const valueSize = 100;

function randomString(length) {
  const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let res = '';
  for (let i = 0; i < length; i++) {
    res += charset.charAt(Math.floor(Math.random() * charset.length));
  }
  return res;
}

export const options = {
  scenarios: {
    main_scenario: {
      executor: 'ramping-arrival-rate',

      timeUnit: '1s',
      preAllocatedVUs: 2000,
      maxVUs: 10000,
      stages: [
        { duration: '15s', target: 50000 }, // ramp up to 80k RPS
        { duration: '1m', target: 50000 },  // stay at 80k RPS
        { duration: '15s', target: 0 },      // ramp down
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
    'http_req_duration{method:PUT}': ['p(95)<50'],
    'http_req_duration{method:DELETE}': ['p(95)<50'],
  },
};


export default function () {
  const key = randomString(keySize);
  const value = randomString(valueSize);
  const url = `${baseURL}/${key}`;

  const putRes = http.put(url, value, { tags: { name: 'PUT /v1/:key', method: 'PUT' } });
  check(putRes, {
    'PUT status is 201': (r) => r.status === 201,
  });

  const delRes = http.del(url, null, { tags: { name: 'DELETE /v1/:key', method: 'DELETE' } });
  check(delRes, {
    'DELETE status is 200': (r) => r.status === 200,
  });
}
