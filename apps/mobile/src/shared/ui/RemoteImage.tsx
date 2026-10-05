import { Image, type ImageProps } from "expo-image";
import { api } from "../../api/client";
import { useTheme } from "../../theme";

/** Loads an API photo (signed relative URL) with disk/memory caching and a soft fade-in. */
export function RemoteImage({ path, style, ...props }: Omit<ImageProps, "source"> & { path: string | undefined }) {
  const { colors } = useTheme();
  return (
    <Image
      source={path ? { uri: api.mediaUrl(path) } : undefined}
      transition={180}
      contentFit="cover"
      cachePolicy="memory-disk"
      recyclingKey={path}
      style={[{ backgroundColor: colors.surfaceMuted }, style]}
      {...props}
    />
  );
}
