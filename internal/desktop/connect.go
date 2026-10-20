package desktop

import (
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/egoist/mygo/ui"
)

// Steps of the Cloudflare sign-in.
const (
	stepSignIn = iota
	stepBucket
)

// openConnect opens the add-or-edit connection page.
func (a *App) openConnect(id string) {
	state := &a.connect
	state.open = true
	state.editing = id
	state.err = nil
	state.notice = ""
	state.testing = false
	state.busy = false
	state.signIn = service.CloudflareSignIn{}
	state.accounts = nil
	state.buckets = nil
	state.step = stepSignIn

	if id == "" {
		state.profile = core.Profile{ProviderType: "r2", CredentialSource: "keychain"}
		state.name = ""
		state.accountID = ""
		state.bucket = ""
		state.jurisdiction = "default"
		state.publicURL = ""
		state.readOnly = false
		state.credentialSource = "keychain"
		state.accessKeyID = ""
		state.secretKey = ""
		state.keychainSaved = false
		return
	}
	profile := a.profileOf(id)
	if profile == nil {
		state.open = false
		return
	}
	state.profile = *profile
	state.name = profile.Name
	state.accountID = profile.AccountID
	state.bucket = profile.Bucket
	state.jurisdiction = profile.Jurisdiction
	state.publicURL = profile.PublicURL
	state.readOnly = profile.ReadOnly
	state.credentialSource = profile.CredentialSource
	state.accessKeyID = ""
	state.secretKey = ""
	state.keychainSaved = profile.CredentialSource != "environment"
	if profile.CredentialSource == "cloudflare" {
		a.loadCloudflareStatus(profile.ID)
	}
}

// closeConnect leaves the connection page.
func (a *App) closeConnect() {
	if a.connect.signIn.SessionID != "" {
		session := a.connect.signIn.SessionID
		go func() { _ = a.svc.CancelCloudflareSignIn(a.ctx, session) }()
	}
	a.connect = connectState{}
}

// connectOverlay draws the connection page over the window.
func (a *App) connectOverlay(c *ui.Context) {
	state := &a.connect
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		page := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(a.pal.background).Modal()
		page.Children(func() {
			a.connectHeader(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).MaxWidth(880).Gap(20).Padding(24, 32).Margin(0, ui.Auto).Children(func() {
					a.connectIntro(c)
					a.connectForm(c)
				})
			})
		})
	})
}

// connectHeader carries the title and the close button.
func (a *App) connectHeader(c *ui.Context) {
	state := &a.connect
	title := a.t("connection.titleNew")
	if state.editing != "" {
		title = a.t("connection.titleEdit")
	}
	ui.Row(c).Height(56).PaddingX(24).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, iconDatabase).FontSize(18).TextColor(a.pal.primary)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(15).Bold()
			ui.Text(c, a.t("connection.flow.desc")).FontSize(11.5).TextColor(a.pal.muted)
		})
		if a.iconAction(c, iconX, a.t("connection.flow.close")) {
			a.closeConnect()
		}
	})
}

// connectIntro names the provider.
func (a *App) connectIntro(c *ui.Context) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, a.t("connection.provider")).FontSize(11.5).Bold().TextColor(a.pal.faint)
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			a.providerTile(c, "r2", 28)
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, a.t("connection.r2")).FontSize(13).Bold()
				ui.Text(c, a.t("connection.r2Note")).FontSize(11.5).TextColor(a.pal.muted)
			})
		})
	})
}

// connectForm shows the manual fields or the Cloudflare sign-in.
func (a *App) connectForm(c *ui.Context) {
	state := &a.connect
	cloudflare := state.profile.CredentialSource == "cloudflare"
	if cloudflare {
		a.cloudflareForm(c)
	} else {
		a.manualForm(c)
	}
	if state.err != nil {
		a.planNotice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.err))
	}
	if state.notice != "" {
		a.planNotice(c, a.pal.successBg, a.pal.successText, state.notice)
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.PrimaryButton(c, a.t("common.save")).Clicked() {
			a.saveConnect()
		}
		if ui.Button(c, a.t("connection.test")).Disabled(state.testing).Clicked() {
			a.testConnect()
		}
		if state.testing {
			ui.Spinner(c).Size(14, 14)
		}
		ui.Spacer(c)
		if state.editing != "" {
			if ui.Button(c, a.t("connection.delete")).Clicked() {
				a.askRemoveProfile()
			}
		}
	})
}

// manualForm is the S3 key form.
func (a *App) manualForm(c *ui.Context) {
	state := &a.connect
	a.settingsGroup(c, a.t("connection.manual.facts"), func() {
		a.fieldRow(c, a.t("connection.name"), func() {
			ui.TextInput(c, &state.name).Placeholder(a.t("connection.name")).Width(320)
		})
		a.fieldRow(c, a.t("connection.accountId"), func() {
			input := ui.TextInput(c, &state.accountID).
				Placeholder(a.t("connection.accountId")).Width(320)
			if len(state.accountID) > 0 && len(state.accountID) != 32 {
				input.Border(1, a.pal.destructive)
			}
		})
		a.fieldRow(c, a.t("connection.credentialSource"), func() {
			sources := []string{"keychain", "environment"}
			labels := []string{a.t("connection.keychain"), a.t("connection.environment")}
			index := 0
			for i, name := range sources {
				if name == state.credentialSource {
					index = i
				}
			}
			seg := ui.Segmented(c, &index, labels...).Label(a.t("connection.credentialSource")).Width(300)
			if seg.Changed() && index >= 0 && index < len(sources) {
				state.credentialSource = sources[index]
			}
		})
		if state.credentialSource == "keychain" {
			a.fieldRow(c, a.t("connection.accessKeyId"), func() {
				placeholder := a.t("connection.accessKeyId")
				if state.keychainSaved && state.editing != "" {
					placeholder = a.t("connection.savedInKeychain")
				}
				ui.TextInput(c, &state.accessKeyID).Placeholder(placeholder).Width(320)
			})
			a.fieldRow(c, a.t("connection.secretAccessKey"), func() {
				placeholder := a.t("connection.secretAccessKey")
				if state.keychainSaved && state.editing != "" {
					placeholder = a.t("connection.savedInKeychain")
				}
				ui.TextInput(c, &state.secretKey).Placeholder(placeholder).Password().Width(320)
			})
			ui.Row(c).Padding(0, 14).Children(func() {
				ui.Text(c, a.t("connection.keychainHint")).FontSize(11).TextColor(a.pal.faint)
			})
		} else {
			ui.Row(c).Padding(0, 14).Children(func() {
				ui.Text(c, a.t("connection.environmentHint")).FontSize(11).TextColor(a.pal.faint)
			})
		}
		a.fieldRow(c, a.t("connection.jurisdiction"), func() {
			options := []string{"default", "eu", "fedramp"}
			current := state.jurisdiction
			if current == "" {
				current = "default"
			}
			select_ := ui.Select(c, &current, options).Label(a.t("connection.jurisdiction")).Width(200)
			if select_.Changed() {
				state.jurisdiction = current
			}
		})
		a.fieldRow(c, a.t("connection.bucket"), func() {
			ui.TextInput(c, &state.bucket).Placeholder(a.t("connection.bucket")).Width(320)
		})
		ui.Row(c).Padding(0, 14).Children(func() {
			ui.Text(c, a.t("connection.bucketHint")).FontSize(11).TextColor(a.pal.faint)
		})
		a.fieldRow(c, a.t("connection.publicUrl"), func() {
			ui.TextInput(c, &state.publicURL).Placeholder("https://").Width(320)
		})
		a.fieldRow(c, a.t("connection.readOnly"), func() {
			ui.Switch(c, &state.readOnly).Label(a.t("connection.readOnly"))
		})
		ui.Row(c).Padding(0, 14).Children(func() {
			ui.Text(c, a.t("connection.readOnlyHint")).FontSize(11).TextColor(a.pal.faint)
		})
	})
	if state.editing != "" {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, a.t("connection.cloudflare.continue")).Clicked() {
				state.profile.CredentialSource = "cloudflare"
				state.profile.Bucket = state.bucket
				a.beginCloudflareSignIn()
			}
		})
	}
}

// cloudflareForm is the sign-in flow of a Cloudflare connection.
func (a *App) cloudflareForm(c *ui.Context) {
	state := &a.connect
	a.settingsGroup(c, a.t("connection.cloudflare.authorization"), func() {
		switch {
		case state.signInBusy:
			ui.Column(c).Padding(16).Gap(10).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, a.t("connection.cloudflare.waiting")).FontSize(12.5)
				})
				ui.Text(c, a.t("connection.cloudflare.waitingHint")).FontSize(11.5).
					TextColor(a.pal.muted)
				if state.signIn.AuthURL != "" {
					ui.Text(c, a.t("connection.cloudflare.authUrl")).FontSize(11).
						TextColor(a.pal.faint)
					ui.Text(c, state.signIn.AuthURL).FontSize(11).Font("monospace").
						TextColor(a.pal.muted).Selectable().SingleLine()
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, a.t("connection.cloudflare.openBrowser")).Clicked() {
							a.openExternally(state.signIn.AuthURL)
						}
						if ui.Button(c, a.t("connection.cloudflare.cancel")).Clicked() {
							a.cancelSignIn()
						}
					})
				}
			})
		case len(state.accounts) > 0:
			a.fieldRow(c, a.t("connection.cloudflare.account"), func() {
				options := make([]string, 0, len(state.accounts))
				for _, account := range state.accounts {
					options = append(options, account.Name)
				}
				current := options[0]
				for i, account := range state.accounts {
					if account.ID == state.accountID2 {
						current = options[i]
					}
				}
				select_ := ui.Select(c, &current, options).
					Label(a.t("connection.cloudflare.account")).Width(260)
				if select_.Changed() {
					for _, account := range state.accounts {
						if account.Name == current && account.ID != state.accountID2 {
							state.accountID2 = account.ID
							state.buckets = nil
							a.loadCloudflareBuckets()
							return
						}
					}
				}
			})
			a.fieldRow(c, a.t("connection.cloudflare.bucket"), func() {
				if len(state.buckets) == 0 {
					ui.Text(c, a.t("connection.cloudflare.noBuckets")).FontSize(12).
						TextColor(a.pal.muted)
					return
				}
				options := make([]string, 0, len(state.buckets))
				for _, bucket := range state.buckets {
					options = append(options, bucket.Name)
				}
				current := state.bucket2
				if current == "" {
					current = options[0]
				}
				select_ := ui.Select(c, &current, options).
					Label(a.t("connection.cloudflare.bucket")).Width(260)
				if select_.Changed() {
					state.bucket2 = current
				}
			})
			a.fieldRow(c, a.t("connection.name"), func() {
				ui.TextInput(c, &state.name).Placeholder(a.t("connection.name")).Width(320)
			})
			a.fieldRow(c, a.t("connection.publicUrl"), func() {
				ui.TextInput(c, &state.publicURL).Placeholder("https://").Width(320)
			})
			a.fieldRow(c, a.t("connection.cloudflare.readOnly"), func() {
				ui.Switch(c, &state.readOnly2).Label(a.t("connection.cloudflare.readOnly"))
			})
			ui.Row(c).Padding(0, 14).Children(func() {
				ui.Text(c, a.t("connection.cloudflare.tokenHint")).FontSize(11).
					TextColor(a.pal.faint)
			})
			ui.Row(c).Gap(8).Padding(0, 14).Children(func() {
				if ui.PrimaryButton(c, a.t("connection.cloudflare.connect")).Clicked() {
					a.connectCloudflare()
				}
				if ui.Button(c, a.t("connection.cloudflare.useKeys")).Clicked() {
					state.profile.CredentialSource = "keychain"
					state.accounts = nil
					state.buckets = nil
				}
			})
		case state.editing != "" && state.status.Authorized:
			ui.Column(c).Padding(16).Gap(8).Children(func() {
				ui.Text(c, a.t("connection.cloudflare.authorized")).FontSize(12.5).Bold()
				ui.Text(c, a.tf("connection.cloudflare.authorizedNote",
					"token", state.status.TokenName)).FontSize(11.5).TextColor(a.pal.muted)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, a.t("connection.cloudflare.reauthorize")).Clicked() {
						a.reauthorizeCloudflare()
					}
					if ui.Button(c, a.t("connection.cloudflare.disconnect")).Clicked() {
						a.askDisconnectCloudflare()
					}
				})
			})
		default:
			ui.Column(c).Padding(16).Gap(8).Children(func() {
				ui.Text(c, a.t("connection.cloudflare.signInDesc")).FontSize(12).
					TextColor(a.pal.muted)
				ui.Text(c, a.t("connection.cloudflare.prereq")).FontSize(11.5).
					TextColor(a.pal.faint)
				if ui.PrimaryButton(c, a.t("connection.cloudflare.continue")).Clicked() {
					a.beginCloudflareSignIn()
				}
			})
		}
	})
	if state.signInErr != nil {
		a.planNotice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.signInErr))
	}
}

// fieldRow is a labelled row of the connection form.
func (a *App) fieldRow(c *ui.Context, label string, control func()) {
	ui.Row(c).Padding(10, 14).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12.5).Grow(1)
		control()
	})
}

// saveConnect stores the connection.
func (a *App) saveConnect() {
	state := &a.connect
	profile := state.profile
	profile.Name = strings.TrimSpace(state.name)
	profile.AccountID = strings.TrimSpace(state.accountID)
	profile.Bucket = strings.TrimSpace(state.bucket)
	profile.Jurisdiction = state.jurisdiction
	profile.PublicURL = strings.TrimSpace(state.publicURL)
	profile.ReadOnly = state.readOnly
	if profile.CredentialSource != "cloudflare" {
		profile.CredentialSource = state.credentialSource
	}
	if profile.Name == "" {
		state.err = core.Invalid("Enter a connection name")
		return
	}
	state.busy = true
	state.err = nil
	input := service.ProfileInput{
		Profile: profile, AccessKeyID: state.accessKeyID, SecretAccessKey: state.secretKey,
	}
	go func() {
		saved, err := a.svc.SaveProfile(a.ctx, input)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.err = err
				return
			}
			a.closeConnect()
			a.loadProfiles()
			if a.prefs.Quick.ProfileID == "" {
				a.prefs.Quick.ProfileID = saved.ID
				a.savePrefs()
			}
			a.toastCurrent(a.t("connection.saved"))
		})
	}()
}

// testConnect checks the settings without saving them.
func (a *App) testConnect() {
	state := &a.connect
	profile := state.profile
	profile.Name = strings.TrimSpace(state.name)
	profile.AccountID = strings.TrimSpace(state.accountID)
	profile.Bucket = strings.TrimSpace(state.bucket)
	profile.Jurisdiction = state.jurisdiction
	profile.PublicURL = strings.TrimSpace(state.publicURL)
	profile.ReadOnly = state.readOnly
	if profile.CredentialSource != "cloudflare" {
		profile.CredentialSource = state.credentialSource
	}
	if profile.Name == "" {
		state.err = core.Invalid("Enter a connection name")
		return
	}
	state.testing = true
	state.err = nil
	state.notice = ""
	input := service.ProfileInput{
		Profile: profile, AccessKeyID: state.accessKeyID, SecretAccessKey: state.secretKey,
	}
	go func() {
		buckets, err := a.svc.TestDraft(a.ctx, input)
		a.update(func() {
			state.testing = false
			if err != nil {
				state.err = err
				return
			}
			state.notice = a.tn("connection.testOk", len(buckets), "count", len(buckets))
		})
	}()
}

// askRemoveProfile confirms before a connection is removed.
func (a *App) askRemoveProfile() {
	state := &a.connect
	id := state.editing
	name := state.name
	a.confirm = confirmState{
		open:        true,
		title:       a.tf("connection.deleteTitle", "name", name),
		message:     a.t("connection.deleteDesc"),
		confirm:     a.t("connection.delete"),
		destructive: true,
		action: func() {
			go func() {
				err := a.svc.RemoveProfile(a.ctx, id, true)
				a.update(func() {
					if err != nil {
						a.toastCurrent(a.errorText(err))
						return
					}
					a.closeConnect()
					a.forgetProfile(id)
					a.loadProfiles()
					a.toastCurrent(a.t("connection.deleted"))
				})
			}()
		},
	}
}

// beginCloudflareSignIn starts the browser authorization.
func (a *App) beginCloudflareSignIn() {
	state := &a.connect
	state.signInBusy = true
	state.signInErr = nil
	state.accounts = nil
	state.buckets = nil
	a.svc.OnCloudflareSignIn = func(update service.CloudflareSignIn) {
		a.update(func() {
			if !a.connect.open {
				return
			}
			a.connect.signIn = update
			if update.Error != nil {
				a.connect.signInBusy = false
				a.connect.signInErr = update.Error
				return
			}
			if len(update.Accounts) > 0 {
				a.connect.accounts = update.Accounts
				if a.connect.accountID2 == "" {
					a.connect.accountID2 = update.Accounts[0].ID
					a.loadCloudflareBuckets()
				}
			}
		})
	}
	go func() {
		session, err := a.svc.StartCloudflareSignIn(a.ctx)
		a.update(func() {
			state.signIn = session
			if err != nil {
				state.signInBusy = false
				state.signInErr = err
				return
			}
			a.openExternally(session.AuthURL)
		})
	}()
}

// loadCloudflareBuckets reads the buckets of the authorized account.
func (a *App) loadCloudflareBuckets() {
	state := &a.connect
	if state.signIn.SessionID == "" || state.accountID2 == "" {
		return
	}
	session, account := state.signIn.SessionID, state.accountID2
	go func() {
		buckets, err := a.svc.CloudflareBuckets(a.ctx, session, account)
		a.update(func() {
			if a.connect.signIn.SessionID != session || a.connect.accountID2 != account {
				return
			}
			if err != nil {
				a.connect.signInErr = err
				return
			}
			a.connect.buckets = buckets
			if a.connect.bucket2 == "" && len(buckets) > 0 {
				a.connect.bucket2 = buckets[0].Name
			}
			a.connect.step = stepBucket
		})
	}()
}

// cancelSignIn stops a pending authorization.
func (a *App) cancelSignIn() {
	state := &a.connect
	session := state.signIn.SessionID
	state.signInBusy = false
	state.signIn = service.CloudflareSignIn{}
	go func() {
		if session != "" {
			_ = a.svc.CancelCloudflareSignIn(a.ctx, session)
		}
	}()
}

// connectCloudflare saves a connection from the authorization.
func (a *App) connectCloudflare() {
	state := &a.connect
	bucket := state.bucket2
	if bucket == "" && len(state.buckets) > 0 {
		bucket = state.buckets[0].Name
	}
	if bucket == "" {
		state.signInErr = core.Invalid("Choose a bucket")
		return
	}
	name := strings.TrimSpace(state.name)
	if name == "" {
		name = bucket
	}
	input := service.CloudflareConnection{
		SessionID: state.signIn.SessionID,
		ProfileID: state.editing,
		Name:      name,
		AccountID: state.accountID2,
		Bucket:    bucket,
		PublicURL: strings.TrimSpace(state.publicURL),
		ReadOnly:  state.readOnly2,
	}
	state.busy = true
	go func() {
		profile, err := a.svc.ConnectCloudflare(a.ctx, input)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.signInErr = err
				return
			}
			a.closeConnect()
			a.loadProfiles()
			a.toastCurrent(a.tf("connection.cloudflare.connected", "bucket", profile.Bucket))
		})
	}()
}

// loadCloudflareStatus reads the stored authorization of a connection.
func (a *App) loadCloudflareStatus(profileID string) {
	go func() {
		status, err := a.svc.CloudflareStatus(a.ctx, profileID)
		if err != nil {
			return
		}
		a.update(func() {
			if a.connect.editing == profileID {
				a.connect.status = status
			}
		})
	}()
}

// reauthorizeCloudflare renews the R2 token of a connection.
func (a *App) reauthorizeCloudflare() {
	state := &a.connect
	id := state.editing
	state.busy = true
	go func() {
		_, err := a.svc.ReauthorizeCloudflare(a.ctx, id)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.err = err
				return
			}
			a.toastCurrent(a.t("connection.cloudflare.reauthorized"))
		})
	}()
}

// askDisconnectCloudflare confirms before the R2 token is deleted.
func (a *App) askDisconnectCloudflare() {
	state := &a.connect
	id := state.editing
	name := state.name
	a.confirm = confirmState{
		open:        true,
		title:       a.tf("connection.cloudflare.disconnectTitle", "name", name),
		message:     a.t("connection.cloudflare.disconnectDesc"),
		confirm:     a.t("connection.cloudflare.disconnect"),
		destructive: true,
		action: func() {
			go func() {
				err := a.svc.DisconnectCloudflare(a.ctx, id, true)
				a.update(func() {
					if err != nil {
						a.toastCurrent(a.errorText(err))
						return
					}
					a.closeConnect()
					a.loadProfiles()
					a.toastCurrent(a.t("connection.cloudflare.disconnected"))
				})
			}()
		},
	}
}
