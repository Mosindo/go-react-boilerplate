import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { ApiError } from "../../api/client";
import { ThemeProvider } from "../../shared/ui";
import OnboardingScreen from "../OnboardingScreen";

const mockGetOwn = jest.fn();
const mockSave = jest.fn();
const mockSavePreferences = jest.fn();

jest.mock("../../hooks/useAuth", () => ({
  useAuth: () => ({ user: { id: "u1", photoCount: 0 }, refreshUser: jest.fn(), signOut: jest.fn(), accessToken: "t" })
}));
jest.mock("../../hooks/useDeviceLocation", () => ({ useDeviceLocation: () => ({ state: "idle", message: null, share: jest.fn() }) }));
jest.mock("../../api/platform", () => ({
  profileApi: {
    getOwn: (...a: unknown[]) => mockGetOwn(...a),
    save: (...a: unknown[]) => mockSave(...a),
    savePreferences: (...a: unknown[]) => mockSavePreferences(...a),
    interests: async () => []
  },
  photoApi: { list: async () => [] }
}));

function renderScreen() {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false }, mutations: { gcTime: Infinity } } });
  render(
    <ThemeProvider forceScheme="light">
      <QueryClientProvider client={client}>
        <OnboardingScreen />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

const yearsAgo = (years: number) => {
  const d = new Date();
  d.setFullYear(d.getFullYear() - years);
  d.setDate(d.getDate() - 3);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getDate())}${pad(d.getMonth() + 1)}${d.getFullYear()}`;
};

beforeEach(() => {
  mockGetOwn.mockReset().mockRejectedValue(new ApiError(404, "profile not found"));
  mockSave.mockReset().mockResolvedValue({});
  mockSavePreferences.mockReset().mockResolvedValue({});
});

describe("OnboardingScreen", () => {
  it("refuses people under 18 before anything is sent to the server", async () => {
    renderScreen();
    fireEvent.changeText(await screen.findByLabelText("Prénom"), "Camille");
    fireEvent.changeText(screen.getByLabelText("Date de naissance"), yearsAgo(17));
    fireEvent.press(screen.getByRole("checkbox", { name: "Femme" }));
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));

    expect(await screen.findByText(/au moins 18 ans/)).toBeTruthy();
    expect(mockSave).not.toHaveBeenCalled();
  });

  it("requires a gender and a first name", async () => {
    renderScreen();
    await screen.findByLabelText("Prénom");
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(await screen.findByText("Indiquez votre prénom.")).toBeTruthy();

    fireEvent.changeText(screen.getByLabelText("Prénom"), "Camille");
    fireEvent.changeText(screen.getByLabelText("Date de naissance"), yearsAgo(30));
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(await screen.findByText("Choisissez une option pour votre genre.")).toBeTruthy();
  });

  it("saves profile and preferences when moving from the basics to the photos step", async () => {
    mockGetOwn.mockRejectedValue(new ApiError(404, "profile not found"));
    renderScreen();
    fireEvent.changeText(await screen.findByLabelText("Prénom"), "  Camille ");
    fireEvent.changeText(screen.getByLabelText("Date de naissance"), yearsAgo(30));
    fireEvent.press(screen.getByRole("checkbox", { name: "Femme" }));
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));

    expect(await screen.findByText("Qui souhaitez-vous rencontrer ?")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));

    await waitFor(() => expect(mockSave).toHaveBeenCalledTimes(1));
    expect(mockSave.mock.calls[0]?.[0]).toMatchObject({ firstName: "Camille", gender: "woman" });
    expect(mockSave.mock.calls[0]?.[0].birthDate).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    await waitFor(() => expect(mockSavePreferences).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Vos plus belles photos")).toBeTruthy();
  });

  it("shows a retry state instead of restarting when the profile cannot be loaded", async () => {
    mockGetOwn.mockRejectedValue(new ApiError(0, "offline"));
    renderScreen();
    expect(await screen.findByText("Impossible de charger votre profil.")).toBeTruthy();
  });
});
