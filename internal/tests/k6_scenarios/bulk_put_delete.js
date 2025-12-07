import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL ? `${__ENV.BASE_URL}/v1` : 'http://localhost:8081/v1';
const keySize = 32;
const valueSize = 256;

function randomString(length) {
  const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let res = '';
  for (let i = 0; i < length; i++) {
    res += charset.charAt(Math.floor(Math.random() * charset.length));
  }
  return res;
}

export const options = {
  systemTags: [
    'status', 'method', 'name', 'proto', 'subproto',
    'tls_version', 'error', 'error_code', 'group',
    'check', 'scenario', 'service', 'expected_response'
  ],

  scenarios: {
    main_scenario: {
      executor: 'ramping-arrival-rate',
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 2000,
      stages: [
        { duration: '45s', target: 5000 },
        { duration: '1m', target: 10000 },
        { duration: '15s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
    'http_req_duration{method:GET}': ['p(99)<50'],
    'http_req_duration{method:PUT}': ['p(99)<50'],
    'http_req_duration{method:DELETE}': ['p(99)<50'],
  },
};

export default function () {
  const key = randomString(keySize);
  const value = randomString(valueSize);
  const url = http.url`${baseURL}/${key}`;

  // 1. PUT
  const putRes = http.put(url, value, { tags: { method: 'PUT' } });

  if (
    !check(putRes, {
      'PUT status is 201': (r) => r.status === 201,
    })
  ) {
    return;
  }

  // 2. GET
  const getRes = http.get(url, { tags: { method: 'GET' } });

  check(getRes, {
    'GET status is 200': (r) => r.status === 200,
    'GET value match': (r) => r.body === value,
  });

  // 3. DELETE
  const delRes = http.del(url, null, { tags: { method: 'DELETE' } });
  check(delRes, {
    'DELETE status is 204': (r) => r.status === 204,
  });
}
