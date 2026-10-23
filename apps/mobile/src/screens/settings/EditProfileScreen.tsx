import React, { useEffect, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { saveProfile } from "../../api";
import {
  BirthDateFields,
  GenderPicker,
  InterestPicker,
  LocationField
} from "../../features/ProfileFields";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import {
  draftFromProfile,
  draftToInput,
  emptyDraft,
  hasErrors,
  validateDraft,
  type DraftErrors,
  type ProfileDraft
} from "../../lib/profileDraft";
import { useProfile } from "../../queries";
import { queryKeys } from "../../session/queryKeys";
import { Button, ErrorState, LoadingState, Screen, Text, TextField, useToast } from "../../ui";
import type { MainStackParams } from "../navTypes";

type Props = NativeStackScreenProps<MainStackParams, "EditProfile">;

export default function EditProfileScreen({ navigation }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const profile = useProfile();
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [ready, setReady] = useState(false);
  const [locationTouched, setLocationTouched] = useState(false);
  const [errors, setErrors] = useState<DraftErrors>({});

  useEffect(() => {
    if (profile.data && !ready) {
      setDraft(draftFromProfile(profile.data));
      setReady(true);
    }
  }, [profile.data, ready]);

  const update = (patch: Partial<ProfileDraft>) =>
    setDraft((current) => ({ ...current, ...patch }));

  const save = useMutation({
    mutationFn: () => {
      const input = draftToInput(draft);
      if (!input) throw new Error("invalid draft");
      // Untouched location: omit the coordinates so the stored (rounded) position is kept.
      if (profile.data?.hasLocation && !locationTouched) {
        delete input.latitude;
        delete input.longitude;
      }
      return saveProfile(input);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      toast.success(t.common.saved);
      navigation.goBack();
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  if (profile.isPending) return <LoadingState />;
  if (profile.isError)
    return <ErrorState onRetry={() => profile.refetch()} title={t.profile.loadError} />;

  const submit = () => {
    const next = validateDraft(draft);
    setErrors(next);
    if (!hasErrors(next)) save.mutate();
  };

  return (
    <Screen scroll testID="edit-profile-screen">
      <TextField
        autoCapitalize="words"
        error={errors.firstName}
        label={t.onboarding.firstName}
        maxLength={40}
        onChangeText={(firstName) => update({ firstName })}
        testID="edit-first-name"
        value={draft.firstName}
      />
      <BirthDateFields
        day={draft.day}
        error={errors.birth}
        month={draft.month}
        onChange={(part, value) => update({ [part]: value })}
        year={draft.year}
      />
      <GenderPicker
        error={errors.gender}
        label={t.onboarding.iAm}
        onChange={(gender) => update({ gender })}
        testPrefix="edit-gender"
        value={draft.gender}
      />
      <TextField
        autoCapitalize="words"
        error={errors.city}
        label={t.onboarding.city}
        maxLength={80}
        onChangeText={(city) => update({ city })}
        testID="edit-city"
        value={draft.city}
      />
      <LocationField
        onChange={(location) => {
          setLocationTouched(true);
          update({ location });
        }}
        value={draft.location}
      />
      <TextField
        counter={`${draft.bio.length}/500`}
        error={errors.bio}
        label={t.onboarding.bio}
        multiline
        onChangeText={(bio) => update({ bio })}
        placeholder={t.onboarding.bioPlaceholder}
        testID="edit-bio"
        value={draft.bio}
      />
      <Text variant="label">{t.onboarding.interestsTitle}</Text>
      <InterestPicker onChange={(interests) => update({ interests })} value={draft.interests} />
      <Button
        label={t.common.save}
        loading={save.isPending}
        onPress={submit}
        testID="edit-save-button"
      />
    </Screen>
  );
}
