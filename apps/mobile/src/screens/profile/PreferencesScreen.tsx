import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { Gender, MyProfile } from "../../api/types";
import { AudienceField, RangeFields } from "../../components/ProfileFields";
import { useMyProfile, useUpdatePreferences } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { LoadingView, showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Text } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export default function PreferencesScreen(props: Props) {
  const profile = useMyProfile();
  if (!profile.data) {
    return <LoadingView />;
  }
  return <PreferencesForm {...props} profile={profile.data} />;
}

function PreferencesForm({ navigation, profile }: Props & { profile: MyProfile }) {
  const update = useUpdatePreferences();
  const [audience, setAudience] = useState<Gender[]>(profile.preferences.interestedIn);
  const [range, setRange] = useState({
    minAge: profile.preferences.minAge,
    maxAge: profile.preferences.maxAge,
    distance: profile.preferences.maxDistanceKm
  });
  return (
    <Screen
      footer={
        <Button
          disabled={audience.length === 0}
          label="Enregistrer"
          loading={update.isPending}
          onPress={() =>
            update.mutate(
              { interestedIn: audience, minAge: range.minAge, maxAge: range.maxAge, maxDistanceKm: range.distance },
              {
                onSuccess: () => {
                  showToast("Critères mis à jour", "success");
                  navigation.goBack();
                }
              }
            )
          }
          testID="preferences-save"
        />
      }
      testID="preferences-screen"
    >
      <ScreenHeader subtitle="La compatibilité est réciproque : vous vous correspondez mutuellement." title="Critères de rencontre" />
      <AudienceField onChange={setAudience} value={audience} />
      <RangeFields distance={range.distance} maxAge={range.maxAge} minAge={range.minAge} onChange={setRange} />
      {update.isError ? <Text tone="danger">{errorMessage(update.error)}</Text> : null}
    </Screen>
  );
}
