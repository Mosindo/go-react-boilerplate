import React, { useCallback, useEffect, useState } from 'react';
import { Image, Platform, StyleSheet, View, type ImageStyle, type StyleProp } from 'react-native';
import { api } from '../../api/http';
import { useTheme } from '../../theme';
import { tokenStore } from '../../store/tokenStore';
import { createImageCache } from './imageCache';

/** Photo urls are API paths that need the Authorization header. */
export type AuthImageProps = {
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  resizeMode?: 'cover' | 'contain';
  accessibilityLabel?: string;
  testID?: string;
  fallback?: React.ReactNode;
};

// Web cannot send headers from <img>: download through the API client (with refresh) -> blob URL.
const webCache = createImageCache(async (path) => URL.createObjectURL(await api.blob(path)));

/** Drop the cached blob (e.g. after an upload replaced the content of a url). */
export function invalidateAuthImage(path: string): void {
  webCache.invalidate(path);
}

function Placeholder({
  style,
  children,
}: {
  style?: StyleProp<ImageStyle>;
  children?: React.ReactNode;
}) {
  const { colors } = useTheme();
  return (
    <View style={[style as object, styles.placeholder, { backgroundColor: colors.skeleton }]}>
      {children}
    </View>
  );
}

function WebAuthImage({
  path,
  style,
  resizeMode,
  accessibilityLabel,
  testID,
  fallback,
}: AuthImageProps & { path: string }) {
  const [src, setSrc] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    let alive = true;
    setSrc(null);
    setFailed(false);
    webCache.load(path).then(
      (url) => alive && setSrc(url),
      () => alive && setFailed(true),
    );
    return () => {
      alive = false;
    };
  }, [path]);
  if (failed) return <Placeholder style={style}>{fallback}</Placeholder>;
  if (!src) return <Placeholder style={style} />;
  return (
    <Image
      testID={testID}
      source={{ uri: src }}
      style={style}
      resizeMode={resizeMode}
      accessibilityLabel={accessibilityLabel}
      accessible={!!accessibilityLabel}
      accessibilityRole="image"
    />
  );
}

function NativeAuthImage({
  path,
  style,
  resizeMode,
  accessibilityLabel,
  testID,
  fallback,
}: AuthImageProps & { path: string }) {
  const [token, setToken] = useState(() => tokenStore.peek()?.accessToken ?? null);
  const [retried, setRetried] = useState(false);
  const [failed, setFailed] = useState(false);
  const onError = useCallback(() => {
    if (retried) {
      setFailed(true);
      return;
    }
    setRetried(true);
    // The access token may have expired since the screen was built: refresh once and retry.
    api
      .refreshSession()
      .then(() => setToken(tokenStore.peek()?.accessToken ?? null))
      .catch(() => setFailed(true));
  }, [retried]);
  if (failed) return <Placeholder style={style}>{fallback}</Placeholder>;
  return (
    <Image
      key={token ?? 'none'}
      testID={testID}
      source={{
        uri: api.resolveUrl(path),
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      }}
      style={style}
      resizeMode={resizeMode}
      onError={onError}
      accessibilityLabel={accessibilityLabel}
      accessible={!!accessibilityLabel}
      accessibilityRole="image"
    />
  );
}

function AuthImageImpl(props: AuthImageProps) {
  const { path } = props;
  if (!path) return <Placeholder style={props.style}>{props.fallback}</Placeholder>;
  return Platform.OS === 'web' ? (
    <WebAuthImage {...props} path={path} />
  ) : (
    <NativeAuthImage {...props} path={path} />
  );
}

export const AuthImage = React.memo(AuthImageImpl);

const styles = StyleSheet.create({
  placeholder: { alignItems: 'center', justifyContent: 'center', overflow: 'hidden' },
});
