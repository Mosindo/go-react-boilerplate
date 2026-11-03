import React, { useEffect, useState } from "react";
import { Image, Platform, type ImageStyle, type StyleProp } from "react-native";
import { API_URL } from "../config";
import { getTokens } from "../auth/session";
import { useTheme } from "../theme";

const webCache = new Map<string, string>();

type Props = {
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  label?: string;
};

/**
 * Photos are private: they need the bearer token. Native <Image> supports
 * request headers; on web we fetch a blob and use an object URL.
 */
export function AuthImage({ path, style, label }: Props) {
  const theme = useTheme();
  const token = getTokens()?.accessToken ?? null;
  const [webUri, setWebUri] = useState<string | null>(path ? (webCache.get(path) ?? null) : null);

  useEffect(() => {
    if (Platform.OS !== "web" || !path || webCache.has(path)) {
      return;
    }
    let active = true;
    fetch(`${API_URL}${path}`, { headers: { Authorization: `Bearer ${token ?? ""}` } })
      .then((response) => (response.ok ? response.blob() : Promise.reject(new Error("photo"))))
      .then((blob) => {
        const url = URL.createObjectURL(blob);
        webCache.set(path, url);
        if (active) {
          setWebUri(url);
        }
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [path, token]);

  const fallback = { backgroundColor: theme.surfaceAlt };
  if (!path) {
    return <Image accessibilityLabel={label} source={undefined} style={[fallback, style]} />;
  }
  const source =
    Platform.OS === "web"
      ? webUri
        ? { uri: webUri }
        : undefined
      : { uri: `${API_URL}${path}`, headers: { Authorization: `Bearer ${token ?? ""}` } };
  return <Image accessibilityLabel={label} source={source} style={[fallback, style]} resizeMode="cover" />;
}
