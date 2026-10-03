import React, { useEffect, useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage, getOwnProfile, listInterests, queryKeys, saveLocation, saveProfile } from "../api/platform";
import { ProfileFields, draftFromProfile, emptyDraft, validateDraft, type DraftErrors, type ProfileDraft } from "../components/ProfileFields";
import { Screen } from "../components/Screen";
import { useDeviceLocation } from "../components/useDeviceLocation";
import type { RootStackParamList } from "../navigation/types";
import { LoadingView, showToast } from "../shared/feedback";
import { Button, Notice, Text } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;

export default function EditProfileScreen({ navigation }: Props) {
  const client = useQueryClient();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const interests = useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  const location = useDeviceLocation();

  useEffect(() => {
    if (profile.data && !loaded) {
      setDraft(draftFromProfile(profile.data));
      setLoaded(true);
    }
  }, [loaded, profile.data]);

  const save = useMutation({
    mutationFn: async () => {
      const { errors: found, input } = validateDraft(draft);
      setErrors(found);
      if (!input) {
        throw new Error("Please fix the highlighted fields.");
      }
      await saveProfile(input);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: queryKeys.profile });
      showToast("Profile saved.", { tone: "success" });
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e))
  });

  const updateLocation = async () => {
    const found = await location.request();
    if (!found) {
      return;
    }
    try {
      await saveLocation(found.latitude, found.longitude);
      if (found.city) {
        setDraft((d) => ({ ...d, city: found.city }));
      }
      await client.invalidateQueries({ queryKey: queryKeys.profile });
      showToast("Location updated.", { tone: "success" });
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  if (!loaded) {
    return <LoadingView fullScreen label="Loading…" />;
  }

  return (
    <Screen testID="edit-profile">
      {error ? <Notice description={error} title="Not saved" tone="danger" /> : null}
      <ProfileFields draft={draft} errors={errors} interests={[]} onChange={setDraft} section="basics" />
      <ProfileFields draft={draft} errors={errors} interests={interests.data ?? []} onChange={setDraft} section="story" />
      <View style={{ gap: 8 }}>
        <Text weight="semibold">Location</Text>
        <Text tone="muted" variant="caption">
          Stored rounded to about 1 km. Others only see an approximate distance.
        </Text>
        <Button label={profile.data?.hasLocation ? "Update my location" : "Use my location"} loading={location.status === "loading"} onPress={() => void updateLocation()} size="sm" variant="secondary" />
        {location.status === "denied" ? (
          <Text tone="muted" variant="caption">
            Location permission is off. Enable it in your device settings.
          </Text>
        ) : null}
      </View>
      <Button label="Save changes" loading={save.isPending} onPress={() => save.mutate()} size="lg" testID="edit-save" />
    </Screen>
  );
}
