import { ActivityIndicator, type ActivityIndicatorProps } from "react-native";
import { useTheme } from "../../theme";

export function Loader(props: ActivityIndicatorProps) {
  const { colors } = useTheme();
  return <ActivityIndicator color={colors.primary} accessibilityLabel="Chargement" {...props} />;
}
