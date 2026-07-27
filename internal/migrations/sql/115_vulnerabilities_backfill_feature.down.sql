UPDATE Organization
SET features = array_remove(features, 'vulnerabilities');
