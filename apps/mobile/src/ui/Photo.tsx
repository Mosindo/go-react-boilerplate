import React, { useEffect, useState } from "react";
import { Image as RNImage, Platform, View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { API_BASE_URL, getFreshAccessToken } from "../api/client";
import { useTheme } from "../theme/theme";

type Props = {
  photoId?: string | null;
  size?: "thumb" | "full";
  style?: StyleProp<ViewStyle>;
  contentFit?: "cover" | "contain";
  accessibilityLabel?: string;
};

const webCache = new Map<string, Promise<string>>();

/** Browsers cannot attach an Authorization header to <img>, so the web build fetches a blob. */
function loadWebBlob(url: string): Promise<string> {
  let pending = webCache.get(url);
  if (!pending) {
    pending = (async () => {
      const token = await getFreshAccessToken();
      const res = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
      if (!res.ok) throw new Error(`photo ${res.status}`);
      return URL.createObjectURL(await res.blob());
    })();
    webCache.set(url, pending);
    pending.catch(() => webCache.delete(url));
  }
  return pending;
}

/** Profile photos are private: every request carries the viewer's access token. */
export function Photo({
  photoId,
  size = "full",
  style,
  contentFit = "cover",
  accessibilityLabel,
}: Props) {
  const { colors } = useTheme();
  const url = photoId
    ? `${API_BASE_URL}/photos/${photoId}${size === "thumb" ? "?size=thumb" : ""}`
    : null;
  const [webSrc, setWebSrc] = useState<string | null>(null);
  const [nativeToken, setNativeToken] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setWebSrc(null);
    if (!url) return;
    if (Platform.OS === "web") {
      loadWebBlob(url)
        .then((src) => !cancelled && setWebSrc(src))
        .catch(() => undefined);
    } else {
      getFreshAccessToken().then((t) => !cancelled && setNativeToken(t));
    }
    return () => {
      cancelled = true;
    };
  }, [url]);

  const placeholder = <View style={[{ backgroundColor: colors.surfaceAlt }, style]} />;
  if (!url) return placeholder;

  if (Platform.OS === "web") {
    if (!webSrc) return placeholder;
    return (
      <RNImage
        source={{ uri: webSrc }}
        accessibilityLabel={accessibilityLabel}
        style={style as never}
        resizeMode={contentFit === "contain" ? "contain" : "cover"}
      />
    );
  }

  if (!nativeToken) return placeholder;
  return (
    <Image
      // cacheKey ignores the rotating token: photo ids are immutable, so the disk cache stays valid.
      source={{
        uri: url,
        headers: { Authorization: `Bearer ${nativeToken}` },
        cacheKey: `${photoId}-${size}`,
      }}
      accessibilityLabel={accessibilityLabel}
      style={style as never}
      contentFit={contentFit}
      transition={150}
      cachePolicy="disk"
    />
  );
}
