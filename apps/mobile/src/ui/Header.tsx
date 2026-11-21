import React from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { makeStyles } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type HeaderProps = {
  title?: string;
  back?: boolean;
  right?: React.ReactNode;
  center?: React.ReactNode;
};

export function Header({ title, back = true, right, center }: HeaderProps) {
  const styles = useStyles();
  const navigation = useNavigation();
  return (
    <View style={styles.row}>
      <View style={styles.side}>
        {back && navigation.canGoBack() ? (
          <IconButton icon="chevron-back" accessibilityLabel="Retour" onPress={() => navigation.goBack()} />
        ) : null}
      </View>
      <View style={styles.center}>
        {center ?? (
          <Text variant="heading" numberOfLines={1} accessibilityRole="header">
            {title}
          </Text>
        )}
      </View>
      <View style={[styles.side, styles.right]}>{right}</View>
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  row: { flexDirection: "row", alignItems: "center", minHeight: 52, paddingHorizontal: t.spacing.sm },
  side: { width: 52, alignItems: "flex-start" },
  right: { alignItems: "flex-end" },
  center: { flex: 1, alignItems: "center" }
}));
