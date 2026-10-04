import React, { useState } from "react";
import { getErrorMessage } from "../../api/client";
import type { OwnProfile, Preferences } from "../../api/types";
import { useOwnProfile, useSavePreferences, useSaveProfile } from "../../hooks/useProfile";
import { Button, Card, ErrorState, LoadingState, Screen, SwitchRow, Text, toast } from "../../ui";
import { PreferencesFields } from "./PreferencesFields";

export function PreferencesScreen({ navigation }: { navigation: { goBack: () => void } }) {
  const query = useOwnProfile();
  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return <Form onDone={() => navigation.goBack()} profile={query.data} />;
}

function Form({ profile, onDone }: { profile: OwnProfile; onDone: () => void }) {
  const savePrefs = useSavePreferences();
  const saveProfile = useSaveProfile();
  const [prefs, setPrefs] = useState<Preferences>(profile.preferences);
  const [discoverable, setDiscoverable] = useState(profile.discoverable);
  const [showDistance, setShowDistance] = useState(profile.showDistance);

  async function submit() {
    if (prefs.interestedIn.length === 0) {
      toast.error("Choose at least one option under “Show me”.");
      return;
    }
    try {
      await savePrefs.mutateAsync(prefs);
      if (discoverable !== profile.discoverable || showDistance !== profile.showDistance) {
        await saveProfile.mutateAsync({
          displayName: profile.displayName,
          birthDate: profile.birthDate,
          gender: profile.gender,
          bio: profile.bio,
          city: profile.city,
          interests: profile.interests.map((i) => i.slug),
          showDistance,
          discoverable
        });
      }
      toast.success("Preferences saved");
      onDone();
    } catch (e) {
      toast.error(getErrorMessage(e));
    }
  }

  return (
    <Screen scroll>
      <PreferencesFields onChange={setPrefs} value={prefs} />
      <Text style={{ marginTop: 8 }} weight="semibold">
        Privacy
      </Text>
      <Card>
        <SwitchRow
          description="Turn off to pause your profile. Your matches and chats stay."
          label="Show me in Discover"
          onValueChange={setDiscoverable}
          value={discoverable}
        />
        <SwitchRow
          description="Others see an approximate distance when both of you share it."
          label="Show my distance"
          onValueChange={setShowDistance}
          value={showDistance}
        />
      </Card>
      <Button label="Save preferences" loading={savePrefs.isPending || saveProfile.isPending} onPress={submit} />
    </Screen>
  );
}
