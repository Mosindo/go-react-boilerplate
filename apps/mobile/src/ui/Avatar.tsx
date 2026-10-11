import { mediaUrl } from '../api/client';
import { initials } from '../lib/format';
import { RemoteImage } from './RemoteImage';

interface AvatarProps {
  /** Relative signed URL from the API (or absolute). */
  url: string | null | undefined;
  name: string;
  size?: number;
}

export function Avatar({ url, name, size = 48 }: AvatarProps) {
  return (
    <RemoteImage
      uri={url ? mediaUrl(url) : null}
      fallbackText={initials(name)}
      accessibilityLabel={name}
      style={{ width: size, height: size, borderRadius: size / 2 }}
    />
  );
}
