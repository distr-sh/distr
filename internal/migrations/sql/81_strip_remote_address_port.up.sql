-- Strip ephemeral ports from IPv4 addresses (e.g. "192.168.1.1:54321" -> "192.168.1.1")
UPDATE ArtifactVersionPull
SET remote_address = regexp_replace(remote_address, ':\d+$', '')
WHERE remote_address ~ '^\d+\.\d+\.\d+\.\d+:\d+$';

-- Strip ephemeral ports from IPv6 addresses (e.g. "[::1]:54321" -> "::1")
UPDATE ArtifactVersionPull
SET remote_address = regexp_replace(remote_address, '^\[(.+)\]:\d+$', '\1')
WHERE remote_address ~ '^\[.+\]:\d+$';
