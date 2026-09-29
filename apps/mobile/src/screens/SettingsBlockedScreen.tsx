import React, { useState } from "react";
import { View } from "react-native";
import { useBlocks, useUnblockUser } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import { EmptyView } from "../shared/feedback/EmptyView";
import { showToast } from "../shared/feedback/toast";
import { Avatar } from "../shared/ui/Avatar";
import { Button } from "../shared/ui/Button";
import { ListItem } from "../shared/ui/ListItem";
import { Notice } from "../shared/ui/Notice";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";
import { ProfileQueryGate } from "./ProfileQueryGate";
import { ProfileSubScreen } from "./ProfileSubScreen";

const makeStyles = (t: Theme) => ({ list: { gap: t.spacing.sm } });

export function SettingsBlockedScreen({ onBack }: { onBack: () => void }) {
  const styles = useThemedStyles(makeStyles);
  const query = useBlocks();
  const unblock = useUnblockUser();
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleUnblock(userId: string, name: string) {
    setPendingId(userId);
    setError(null);
    try {
      await unblock.mutateAsync(userId);
      showToast(`${name} was unblocked`);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setPendingId(null);
    }
  }

  return (
    <ProfileQueryGate onBack={onBack} query={query} title="Blocked people">
      {(blocks) => (
        <ProfileSubScreen
          onBack={onBack}
          subtitle="Blocked people cannot see you or message you, and you will not see them."
          testID="settings-blocked-screen"
          title="Blocked people"
        >
          {error ? <Notice title={error} tone="danger" /> : null}
          {blocks.length === 0 ? (
            <EmptyView message="Nobody is blocked. You can block someone from their profile or a chat." title="No blocked people" />
          ) : (
            <View style={styles.list}>
              {blocks.map((block) => (
                <ListItem
                  key={block.userId}
                  leading={<Avatar name={block.firstName} />}
                  subtitle={`Blocked on ${new Date(block.blockedAt).toLocaleDateString()}`}
                  title={block.firstName}
                  trailing={
                    <Button
                      accessibilityLabel={`Unblock ${block.firstName}`}
                      label="Unblock"
                      loading={pendingId === block.userId}
                      onPress={() => void handleUnblock(block.userId, block.firstName)}
                      size="sm"
                      testID={`unblock-${block.userId}`}
                      variant="outline"
                    />
                  }
                />
              ))}
            </View>
          )}
        </ProfileSubScreen>
      )}
    </ProfileQueryGate>
  );
}
