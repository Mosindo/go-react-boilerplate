import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { saveProfile } from "../api/platform";
import { renderWithProviders } from "../test-utils";
import OnboardingScreen from "./OnboardingScreen";

jest.mock("../api/platform", () => ({
  fetchOwnProfile: jest.fn().mockRejectedValue(Object.assign(new Error("not found"), { status: 404 })),
  fetchInterests: jest.fn().mockResolvedValue([]),
  saveProfile: jest.fn().mockResolvedValue({}),
  savePreferences: jest.fn().mockResolvedValue({}),
  saveLocation: jest.fn()
}));
jest.mock("../hooks/useAuth", () => ({ useAuth: () => ({ signOut: jest.fn() }) }));
jest.mock("../api/client", () => {
  const actual = jest.requireActual("../api/client");
  return { ...actual, isApiError: (error: unknown, status?: number) => (error as { status?: number }).status === status };
});

function fillBasics(birth: string, name = "Alice") {
  fireEvent.changeText(screen.getByTestId("onboarding-name"), name);
  fireEvent.changeText(screen.getByTestId("onboarding-birth"), birth);
  fireEvent.press(screen.getByTestId("gender-woman"));
  fireEvent.press(screen.getByTestId("onboarding-next"));
}

describe("OnboardingScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("refuses minors and malformed dates", async () => {
    renderWithProviders(<OnboardingScreen onDone={jest.fn()} />);
    const year = new Date().getFullYear() - 16;
    fillBasics(`01/01/${year}`);
    expect(await screen.findByText("Vous devez avoir au moins 18 ans pour utiliser l'application.")).toBeTruthy();

    fireEvent.changeText(screen.getByTestId("onboarding-birth"), "31/02/1990");
    fireEvent.press(screen.getByTestId("onboarding-next"));
    expect(await screen.findByText("Format attendu : JJ/MM/AAAA.")).toBeTruthy();
    expect(saveProfile).not.toHaveBeenCalled();
  });

  it("saves the profile with an ISO birth date and moves on to preferences", async () => {
    renderWithProviders(<OnboardingScreen onDone={jest.fn()} />);
    fillBasics("14/03/1995");
    await waitFor(() =>
      expect(saveProfile).toHaveBeenCalledWith(expect.objectContaining({ firstName: "Alice", birthDate: "1995-03-14", gender: "woman" }))
    );
    expect(await screen.findByText("Qui cherchez-vous ?")).toBeTruthy();

    fireEvent.press(screen.getByTestId("onboarding-next"));
    expect(await screen.findByText("Choisissez au moins une option.")).toBeTruthy();
  });
});
