import React, { useEffect, useMemo, useState } from "react";
import { Image, View, type ImageStyle, type StyleProp, type ViewStyle } from "react-native";
import { API_BASE_URL, http } from "../api";
import { initialsOf } from "../lib/format";
import { createStyles } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = {
  /** Relative path from the API (`/photos/{id}` or `/photos/{id}?size=thumb`), or null. */
  path: string | null | undefined;
  /** Used for the placeholder initial and accessibility label. */
  name?: string;
  style?: StyleProp<ViewStyle>;
  imageStyle?: StyleProp<ImageStyle>;
  resizeMode?: "cover" | "contain";
  label?: string;
};

const useStyles = createStyles(({ colors }) => ({
  box: {
    backgroundColor: colors.surfaceAlt,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  fill: { position: "absolute", top: 0, left: 0, right: 0, bottom: 0 }
}));

/** Photos are private: the Authorization header travels with the image request. */
export function Photo({ path, name = "", style, imageStyle, resizeMode = "cover", label }: Props) {
  const styles = useStyles();
  const [attempt, setAttempt] = useState(0);
  const [failed, setFailed] = useState(false);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    setFailed(false);
    setLoaded(false);
    setAttempt(0);
  }, [path]);

  const source = useMemo(() => {
    if (!path) return null;
    const token = http.getCachedAccessToken();
    return {
      uri: `${API_BASE_URL}${path}`,
      headers: token ? { Authorization: `Bearer ${token}` } : undefined
    };
    // attempt forces a new source (and a fresh token) after an authorization failure
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path, attempt]);

  const onError = async () => {
    if (attempt === 0) {
      try {
        await http.getFreshAccessToken(0);
        setAttempt(1);
        return;
      } catch {
        // fall through to placeholder
      }
    }
    setFailed(true);
  };

  return (
    <View
      accessibilityIgnoresInvertColors
      accessibilityLabel={label}
      accessibilityRole={label ? "image" : undefined}
      style={[styles.box, style]}
    >
      {!source || failed || !loaded ? (
        <Text variant="heading" tone="muted">
          {!source || failed ? initialsOf(name) : ""}
        </Text>
      ) : null}
      {source && !failed ? (
        <Image
          onError={onError}
          onLoad={() => setLoaded(true)}
          resizeMode={resizeMode}
          source={source}
          style={[styles.fill, imageStyle]}
        />
      ) : null}
    </View>
  );
}
