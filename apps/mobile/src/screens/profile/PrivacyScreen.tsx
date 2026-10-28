import React from "react";
import { Switch, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { saveProfile } from "../../api/profile";
import type { ProfileInput } from "../../api/types";
import { ErrorState, LoadingState, Screen, Text, useToast } from "../../components";
import { useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import { profileToInput } from "../../lib/profileInput";
import { spacing } from "../../theme";

function ToggleRow({ title, description, value, onChange, testID }: { title: string; description: string; value: boolean; onChange: (v: boolean) => void; testID: string }) {
  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
      <View style={{ flex: 1, gap: 2 }}>
        <Text variant="label">{title}</Text>
        <Text variant="caption" tone="muted">
          {description}
        </Text>
      </View>
      <Switch value={value} onValueChange={onChange} accessibilityLabel={title} testID={testID} />
    </View>
  );
}

export function PrivacyScreen() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useOwnProfile();
  const mutation = useMutation({
    mutationFn: (patch: Partial<ProfileInput>) => {
      if (!query.data) throw new Error("Profile not loaded.");
      return saveProfile(profileToInput(query.data, patch));
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      await queryClient.invalidateQueries({ queryKey: keys.discover });
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  if (query.isPending) return <Screen><LoadingState /></Screen>;
  if (query.isError || !query.data) return <Screen><ErrorState message={errorMessage(query.error, "Profile not found.")} onRetry={() => void query.refetch()} /></Screen>;
  const profile = query.data;

  return (
    <Screen testID="privacy-screen">
      <View style={{ gap: spacing.xl }}>
        <ToggleRow
          title="Show my distance"
          description="Others see an approximate distance (rounded), never your exact position."
          value={profile.showDistance}
          onChange={(showDistance) => mutation.mutate({ showDistance })}
          testID="privacy-distance"
        />
        <ToggleRow
          title="Appear in discovery"
          description="Turn off to pause your profile. Your matches and chats stay active."
          value={profile.discoverable}
          onChange={(discoverable) => mutation.mutate({ discoverable })}
          testID="privacy-discoverable"
        />
      </View>
    </Screen>
  );
}
