import React, { useEffect, useState } from "react";
import { Image, Platform, View, type ImageStyle, type StyleProp } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { API_BASE_URL, currentAccessToken, fetchAuthedBlob } from "../api/client";
import { useTheme } from "../theme/theme";

// Photos are protected by the API (auth + block checks), so <img> on web cannot load them directly.
// On web we fetch them with the token once and cache the object URL; native passes headers.
const webCache = new Map<string, Promise<string>>();

function webObjectUrl(path: string): Promise<string> {
  let hit = webCache.get(path);
  if (!hit) {
    hit = fetchAuthedBlob(path).then((blob) => URL.createObjectURL(blob));
    webCache.set(path, hit);
    hit.catch(() => webCache.delete(path));
  }
  return hit;
}

/** Drops cached object URLs (called on logout so one session's photos never leak into the next). */
export function clearImageCache(): void {
  if (Platform.OS !== "web") return;
  for (const p of webCache.values()) void p.then((u) => URL.revokeObjectURL(u)).catch(() => undefined);
  webCache.clear();
}

export function AuthImage({
  path,
  style,
  label
}: {
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  label?: string;
}) {
  const t = useTheme();
  const [webUri, setWebUri] = useState<string | null>(null);
  const [nativeToken, setNativeToken] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let alive = true;
    setFailed(false);
    if (!path) return;
    if (Platform.OS === "web") {
      setWebUri(null);
      webObjectUrl(path)
        .then((u) => alive && setWebUri(u))
        .catch(() => alive && setFailed(true));
    } else {
      void currentAccessToken().then((tok) => alive && setNativeToken(tok));
    }
    return () => {
      alive = false;
    };
  }, [path]);

  const uri = Platform.OS === "web" ? webUri : path && nativeToken ? `${API_BASE_URL}${path}` : null;

  if (!path || failed || !uri) {
    return (
      <View
        accessibilityLabel={label}
        style={[{ backgroundColor: t.surfaceAlt, alignItems: "center", justifyContent: "center" }, style as object]}
      >
        {!path || failed ? <Ionicons name="person" size={32} color={t.textMuted} /> : null}
      </View>
    );
  }
  return (
    <Image
      accessibilityLabel={label}
      source={Platform.OS === "web" ? { uri } : { uri, headers: { Authorization: `Bearer ${nativeToken ?? ""}` } }}
      resizeMode="cover"
      onError={() => setFailed(true)}
      style={style}
    />
  );
}

export function Avatar({ path, size = 48, label }: { path?: string | null; size?: number; label?: string }) {
  return (
    <AuthImage
      path={path}
      label={label}
      style={{ width: size, height: size, borderRadius: size / 2, overflow: "hidden" }}
    />
  );
}
