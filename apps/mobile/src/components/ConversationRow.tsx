import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ConversationSummary } from "../api/models";
import { formatRelativeTime, previewText } from "../lib/dating/format";
import { useTheme } from "../shared/ui/theme";
import { Txt } from "./kit";
import { PersonAvatar } from "./PersonAvatar";

type Props = {
  conversation: ConversationSummary;
  myUserId: string;
  onPress: (conversation: ConversationSummary) => void;
  onLongPress: (conversation: ConversationSummary) => void;
};

function ConversationRowBase({ conversation, myUserId, onPress, onLongPress }: Props) {
  const { colors, radii, spacing } = useTheme();
  const { user, lastMessage, unreadCount } = conversation;
  const unread = unreadCount > 0;
  const preview = lastMessage
    ? `${lastMessage.senderId === myUserId ? "You: " : ""}${previewText(lastMessage.body)}`
    : "Say hello 👋";
  const time = formatRelativeTime(conversation.updatedAt);

  return (
    <Pressable
      testID={`conversation-row-${conversation.id}`}
      accessibilityRole="button"
      accessibilityLabel={`${user.firstName}. ${preview}. ${time}${unread ? `. ${unreadCount} unread` : ""}`}
      accessibilityHint="Opens the conversation. Long press for more options."
      onPress={() => onPress(conversation)}
      onLongPress={() => onLongPress(conversation)}
      delayLongPress={350}
      style={({ pressed }) => [
        styles.row,
        {
          minHeight: 72,
          paddingHorizontal: spacing.lg,
          paddingVertical: spacing.sm,
          gap: spacing.md,
          borderRadius: radii.lg,
          backgroundColor: pressed ? colors.surfaceMuted : "transparent"
        }
      ]}
    >
      <PersonAvatar name={user.firstName} photo={user.photo} size={56} />
      <View style={styles.body}>
        <View style={styles.top}>
          <Txt
            variant="subheading"
            weight={unread ? "bold" : "semibold"}
            numberOfLines={1}
            style={styles.name}
          >
            {user.firstName}
          </Txt>
          <Txt
            variant="caption"
            tone={unread ? "primary" : "subtle"}
            weight={unread ? "bold" : "regular"}
          >
            {time}
          </Txt>
        </View>
        <View style={styles.top}>
          <Txt
            tone={unread ? "default" : "muted"}
            weight={unread ? "semibold" : "regular"}
            numberOfLines={1}
            style={styles.name}
          >
            {preview}
          </Txt>
          {unread ? (
            <View
              style={[
                styles.badge,
                {
                  backgroundColor: colors.primary,
                  minWidth: 22,
                  height: 22,
                  borderRadius: 11
                }
              ]}
            >
              <Txt variant="caption" weight="bold" style={{ color: colors.primaryForeground }}>
                {unreadCount > 99 ? "99+" : String(unreadCount)}
              </Txt>
            </View>
          ) : null}
        </View>
      </View>
    </Pressable>
  );
}

export const ConversationRow = React.memo(ConversationRowBase);

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center" },
  body: { flex: 1, gap: 2 },
  top: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 8
  },
  name: { flex: 1 },
  badge: {
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
