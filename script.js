import http from 'k6/http';
import { sleep } from 'k6';

export let options = {
  vus: 1000,      
  duration: '2m' 
};

export default function () {
  http.get('http://localhost:8080/posts');

  let payload = JSON.stringify({
    title: 'Task from load test',
    description: 'This is a test task'
  });

  let params = { headers: { 'Content-Type': 'application/json' } };
  http.post('http://localhost:8080/posts', payload, params);

  sleep(0.5);
}
