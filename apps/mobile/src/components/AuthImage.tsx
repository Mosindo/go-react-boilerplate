import React, { useCallback, useEffect, useState } from "react";
import { Image, Platform, View, type ImageStyle, type StyleProp } from "react-native";
import { API_BASE_URL, refreshSession, request } from "../api/client";
import { currentTokens } from "../api/tokenStore";
import { useTheme } from "../theme";

// Photos are private: every request carries the Bearer token. Native <Image> can send
// headers; <img> on the web cannot, so there we fetch a blob and use an object URL.
const webCache = new Map<string, string>();
const WEB_CACHE_MAX = 80;

async function loadWebBlobUrl(photoId: string): Promise<string> {
  const cached = webCache.get(photoId);
  if (cached) return cached;
  const res = await request(`/photos/${photoId}`);
  const url = URL.createObjectURL(await res.blob());
  if (webCache.size >= WEB_CACHE_MAX) {
    const oldest = webCache.keys().next().value;
    if (oldest !== undefined) {
      URL.revokeObjectURL(webCache.get(oldest) ?? "");
      webCache.delete(oldest);
    }
  }
  webCache.set(photoId, url);
  return url;
}

type Props = {
  photoId: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  resizeMode?: "cover" | "contain";
  accessibilityLabel?: string;
};

export function AuthImage({ photoId, style, resizeMode = "cover", accessibilityLabel }: Props) {
  const { colors } = useTheme();
  const [webUri, setWebUri] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (Platform.OS !== "web" || !photoId) return;
    let alive = true;
    setFailed(false);
    loadWebBlobUrl(photoId)
      .then((u) => alive && setWebUri(u))
      .catch(() => alive && setFailed(true));
    return () => {
      alive = false;
    };
  }, [photoId]);

  const onNativeError = useCallback(async () => {
    // An expired access token is the usual cause: refresh once, then retry.
    if (attempt === 0 && (await refreshSession())) setAttempt(1);
    else setFailed(true);
  }, [attempt]);

  const placeholder = <View style={[style as object, { backgroundColor: colors.surfaceAlt }]} />;
  if (!photoId || failed) return placeholder;

  if (Platform.OS === "web") {
    if (!webUri) return placeholder;
    return (
      <Image
        source={{ uri: webUri }}
        style={style}
        resizeMode={resizeMode}
        accessibilityLabel={accessibilityLabel}
        accessible={!!accessibilityLabel}
      />
    );
  }

  return (
    <Image
      key={attempt}
      source={{
        uri: `${API_BASE_URL}/photos/${photoId}`,
        headers: { Authorization: `Bearer ${currentTokens()?.accessToken ?? ""}` }
      }}
      style={style}
      resizeMode={resizeMode}
      accessibilityLabel={accessibilityLabel}
      accessible={!!accessibilityLabel}
      onError={onNativeError}
    />
  );
}
