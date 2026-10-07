import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { updateMyLocation } from "../../api/profile";
import {
  draftFromProfile,
  buildProfileInput,
  initialDraft,
  type Draft
} from "../../features/onboarding/draft";
import { InterestPicker } from "../../features/profile/InterestPicker";
import { LocationField } from "../../features/profile/LocationField";
import { geocodeCity } from "../../features/profile/location";
import { GENDER_OPTIONS } from "../../features/profile/options";
import { useMyProfile, useSaveProfile } from "../../hooks/useProfile";
import { strings } from "../../i18n/strings";
import type { ProfileStackParamList } from "../../navigation/types";
import { LoadingView, reportError, showToast } from "../../shared/feedback";
import { BirthDateInput, ChipGroup } from "../../shared/form";
import { FormScreen } from "../../shared/layout";
import { Button, Input, Text, spacing } from "../../shared/ui";
import {
  buildBirthDate,
  validateBio,
  validateBirthDate,
  validateCity,
  validateFirstName
} from "../../utils/validation";

type Props = NativeStackScreenProps<ProfileStackParamList, "EditProfile">;

type FieldErrors = { name?: string | null; birth?: string | null; bio?: string | null; city?: string | null };

export function validateProfileDraft(draft: Draft, now: Date = new Date()): FieldErrors {
  return {
    name: validateFirstName(draft.firstName),
    birth: validateBirthDate(buildBirthDate(draft.birth.day, draft.birth.month, draft.birth.year), now),
    bio: validateBio(draft.bio),
    city: validateCity(draft.city)
  };
}

export default function EditProfileScreen({ navigation }: Props) {
  const query = useMyProfile();
  const save = useSaveProfile();
  const [draft, setDraft] = useState<Draft>(initialDraft);
  const [loaded, setLoaded] = useState(false);
  const [errors, setErrors] = useState<FieldErrors>({});

  useEffect(() => {
    if (!loaded && query.data) {
      setDraft(draftFromProfile(query.data));
      setLoaded(true);
    }
  }, [loaded, query.data]);

  if (!loaded) {
    return (
      <FormScreen edges={["left", "right", "bottom"]}>
        <LoadingView fullScreen />
      </FormScreen>
    );
  }

  const update = (patch: Partial<Draft>) => setDraft((current) => ({ ...current, ...patch }));

  async function submit() {
    const found = validateProfileDraft(draft);
    setErrors(found);
    const input = buildProfileInput(draft);
    if (Object.values(found).some(Boolean) || !input) {
      return;
    }
    try {
      await save.mutateAsync(input);
      let coords = draft.coords;
      if (!coords && draft.city.trim() && draft.city.trim() !== query.data?.city) {
        coords = await geocodeCity(draft.city.trim());
      }
      if (coords) {
        await updateMyLocation(coords.latitude, coords.longitude);
      }
      showToast(strings.profile.saved, "success");
      navigation.goBack();
    } catch (error) {
      reportError(error);
    }
  }

  return (
    <FormScreen
      edges={["left", "right", "bottom"]}
      footer={
        <Button
          label={strings.common.save}
          loading={save.isPending}
          onPress={() => void submit()}
          size="lg"
        />
      }
    >
      <Input
        autoCapitalize="words"
        error={errors.name}
        label={strings.onboarding.firstName}
        maxLength={40}
        onChangeText={(firstName) => update({ firstName })}
        value={draft.firstName}
      />
      <BirthDateInput error={errors.birth} onChange={(birth) => update({ birth })} value={draft.birth} />
      <View style={{ gap: spacing.sm }}>
        <Text tone="secondary" variant="label" weight="semibold">
          {strings.profile.gender}
        </Text>
        <ChipGroup
          onToggle={(gender) => update({ gender })}
          options={GENDER_OPTIONS}
          selected={draft.gender ? [draft.gender] : []}
        />
      </View>
      <Input
        error={errors.bio}
        helperText={strings.profile.bioCount(draft.bio.length, 500)}
        label={strings.profile.bio}
        maxLength={500}
        multiline
        onChangeText={(bio) => update({ bio })}
        placeholder={strings.profile.bioPlaceholder}
        value={draft.bio}
      />
      <View style={{ gap: spacing.sm }}>
        <Text tone="secondary" variant="label" weight="semibold">
          {strings.profile.location}
        </Text>
        <LocationField
          city={draft.city}
          cityError={errors.city}
          coords={draft.coords}
          onCity={(city) => update({ city })}
          onCoords={(coords) => update({ coords })}
        />
      </View>
      <View style={{ gap: spacing.sm }}>
        <Text tone="secondary" variant="label" weight="semibold">
          {strings.profile.interests}
        </Text>
        <InterestPicker onChange={(interests) => update({ interests })} selected={draft.interests} />
      </View>
    </FormScreen>
  );
}
