import {DistrService} from '../client';
import {clientConfig} from './config';

const distr = new DistrService(clientConfig);
const appId = '<docker-application-id>';

const composeFile = `
services:
  my-postgres:
    image: 'postgres:18.4-alpine3.23'
    ports:
      - '5434:5432'
    environment:
      POSTGRES_USER: \${POSTGRES_USER}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
      POSTGRES_DB: \${POSTGRES_DB}
    volumes:
      - 'postgres-data:/var/lib/postgresql/data/'

volumes:
  postgres-data:
`;

const templateFile = `
POSTGRES_USER=some-user # REPLACE THIS
POSTGRES_PASSWORD=some-password # REPLACE THIS
POSTGRES_DB=some-db # REPLACE THIS`;

await distr.createDockerApplicationVersion(appId, '18.4-alpine3.23+2', {
  composeFile,
  templateFile,
});
