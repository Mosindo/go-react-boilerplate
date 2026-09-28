import React, { useState } from "react";
import { errorMessage } from "../api/client";
import type { Preferences } from "../api/types";
import { PreferencesForm } from "../components/PreferencesForm";
import { useMe } from "../hooks/useAuth";
import { useSavePreferences } from "../hooks/useProfile";
import type { MainScreenProps } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Notice } from "../shared/ui";

function PreferencesEditor({ initial, onDone }: { initial: Preferences; onDone: () => void }) {
  const [value, setValue] = useState(initial);
  const save = useSavePreferences();
  const invalid = value.interestedIn.length === 0;
  return (
    <ScreenContainer scroll underHeader>
      {save.isError ? <Notice kind="error" message={errorMessage(save.error)} /> : null}
      <PreferencesForm error={invalid ? "Choose at least one option." : null} onChange={setValue} value={value} />
      <Button
        disabled={invalid}
        label="Save preferences"
        loading={save.isPending}
        onPress={() =>
          save.mutate(value, {
            onSuccess: () => {
              showToast("Preferences saved.", "success");
              onDone();
            }
          })
        }
      />
    </ScreenContainer>
  );
}

export default function PreferencesScreen({ navigation }: MainScreenProps<"Preferences">) {
  const meQuery = useMe();
  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (!meQuery.data) {
    return <ErrorView message={errorMessage(meQuery.error)} onRetry={() => void meQuery.refetch()} />;
  }
  return <PreferencesEditor initial={meQuery.data.preferences} onDone={() => navigation.goBack()} />;
}
