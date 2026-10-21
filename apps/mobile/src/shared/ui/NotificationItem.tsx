import React from "react";
import { StyleSheet, View } from "react-native";
import { formatRelative } from "../../lib/format";
import { spacing, useTheme } from "../../theme";
import { Avatar } from "./Avatar";
import { ListItem } from "./ListItem";

type Props = {
  title: string;
  subtitle: string;
  actorName: string;
  actorThumb: string;
  createdAt: string;
  unread: boolean;
  onPress: () => void;
  testID?: string;
};

export function NotificationItem({ title, subtitle, actorName, actorThumb, createdAt, unread, onPress, testID }: Props) {
  const theme = useTheme();
  return (
    <ListItem
      testID={testID}
      title={title}
      subtitle={`${subtitle} · ${formatRelative(createdAt)}`}
      onPress={onPress}
      left={<Avatar name={actorName} uri={actorThumb} />}
      right={unread ? <View accessibilityLabel="Non lu" style={[styles.dot, { backgroundColor: theme.primary }]} /> : null}
    />
  );
}

const styles = StyleSheet.create({ dot: { width: 10, height: 10, borderRadius: 5, marginLeft: spacing.sm } });
