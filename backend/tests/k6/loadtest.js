import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const PRODUCTS = [
  'iphone_15_pro',
  'iphone_15',
  'samsung_s24_ultra',
  'samsung_s24',
  'sony_wh1000xm5',
  'apple_watch_series9',
  'macbook_air_m2',
  'macbook_pro_m3',
  'ipad_pro_12_9',
  'airpods_pro_2',
];

export const options = {
  stages: [
    { duration: '30s', target: 30 },
    { duration: '1m', target: 80 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.K6_BASE_URL || 'http://inventory-api:8080';

export default function () {
  const product = PRODUCTS[Math.floor(Math.random() * PRODUCTS.length)];
  const payload = JSON.stringify({
    product_id: product,
    quantity: 1,
    request_id: uuidv4(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(`${BASE_URL}/api/v1/reserve`, payload, params);

  check(res, {
    'status is 200, 409, or 429': (r) => r.status === 200 || r.status === 409 || r.status === 429,
    'response time < 1.5s': (r) => r.timings.duration < 1500,
  });

  sleep(0.1);
}