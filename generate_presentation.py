import os
import sys
from reportlab.lib import colors
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, PageBreak
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.pdfgen import canvas

# Widescreen 16:9 slide dimensions (960 x 540 pt)
PAGE_WIDTH = 960
PAGE_HEIGHT = 540
PAGESIZE = (PAGE_WIDTH, PAGE_HEIGHT)

# Sophisticated Theme Colors (Tailored Modern Dark/Tech Palette)
COLOR_BG_DARK = colors.HexColor("#0B0F19")      # Deep rich midnight navy
COLOR_CARD_BG = colors.HexColor("#131B2E")      # Slate 900 card surface
COLOR_CARD_BORDER = colors.HexColor("#23314E")  # Subtle slate border
COLOR_PRIMARY = colors.HexColor("#6366F1")      # Vibrant Indigo
COLOR_PRIMARY_LIGHT = colors.HexColor("#818CF8")# Light Indigo
COLOR_CYAN = colors.HexColor("#06B6D4")         # Cyan Accent
COLOR_EMERALD = colors.HexColor("#10B981")      # Emerald Green
COLOR_AMBER = colors.HexColor("#F59E0B")        # Amber Accent
COLOR_TEXT_MAIN = colors.HexColor("#FFFFFF")    # Pure white text
COLOR_TEXT_MUTED = colors.HexColor("#CBD5E1")   # Slate 300
COLOR_TEXT_DIM = colors.HexColor("#64748B")     # Slate 500
COLOR_PILL_BG = colors.HexColor("#1E293B")      # Slate 800

def draw_slide_background(canvas, doc):
    canvas.saveState()
    # 1. Fill entire slide background
    canvas.setFillColor(COLOR_BG_DARK)
    canvas.rect(0, 0, PAGE_WIDTH, PAGE_HEIGHT, fill=1, stroke=0)
    
    # 2. Top primary accent line
    canvas.setFillColor(COLOR_PRIMARY)
    canvas.rect(0, PAGE_HEIGHT - 4, PAGE_WIDTH, 4, fill=1, stroke=0)
    
    page_num = canvas.getPageNumber()
    if page_num > 1:
        # Top Header Bar
        canvas.setStrokeColor(COLOR_CARD_BORDER)
        canvas.setLineWidth(1)
        canvas.line(40, PAGE_HEIGHT - 52, PAGE_WIDTH - 40, PAGE_HEIGHT - 52)
        
        canvas.setFont("Helvetica-Bold", 10)
        canvas.setFillColor(COLOR_PRIMARY_LIGHT)
        canvas.drawString(40, PAGE_HEIGHT - 36, "DIGITERA HELPDESK")
        
        canvas.setFont("Helvetica", 9)
        canvas.setFillColor(COLOR_TEXT_DIM)
        canvas.drawString(170, PAGE_HEIGHT - 36, "|  Enterprise Full-Stack Architecture")
        
        # Bottom Footer Bar
        canvas.line(40, 42, PAGE_WIDTH - 40, 42)
        canvas.setFont("Helvetica", 8)
        canvas.setFillColor(COLOR_TEXT_DIM)
        canvas.drawString(40, 26, "Confidential & Proprietary • Osama Zaki • 2026")
        canvas.drawRightString(PAGE_WIDTH - 40, 26, f"Slide {page_num} of 7")
    else:
        # Cover slide bottom bar
        canvas.setStrokeColor(COLOR_CARD_BORDER)
        canvas.setLineWidth(1)
        canvas.line(40, 42, PAGE_WIDTH - 40, 42)
        canvas.setFont("Helvetica", 8)
        canvas.setFillColor(COLOR_TEXT_DIM)
        canvas.drawString(40, 26, "Digitera Systems • Technical Project Presentation • 2026")
        canvas.drawRightString(PAGE_WIDTH - 40, 26, "https://github.com/osamazakimohammed/helpdesk")
        
    canvas.restoreState()

def build_pdf(filename="Digitera_Helpdesk_Project_Presentation.pdf"):
    doc = SimpleDocTemplate(
        filename,
        pagesize=PAGESIZE,
        leftMargin=40,
        rightMargin=40,
        topMargin=64,
        bottomMargin=50,
    )

    styles = getSampleStyleSheet()

    # Custom Typography Styles
    title_style = ParagraphStyle(
        'CoverTitle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=30,
        leading=36,
        textColor=COLOR_TEXT_MAIN,
        spaceAfter=10
    )

    subtitle_style = ParagraphStyle(
        'CoverSubtitle',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=14,
        leading=18,
        textColor=COLOR_PRIMARY_LIGHT,
        spaceAfter=25
    )

    section_heading = ParagraphStyle(
        'SectionHeading',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=18,
        leading=22,
        textColor=COLOR_TEXT_MAIN,
        spaceAfter=3
    )

    section_subheading = ParagraphStyle(
        'SectionSubHeading',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=10,
        leading=13,
        textColor=COLOR_TEXT_MUTED,
        spaceAfter=14
    )

    card_title = ParagraphStyle(
        'CardTitle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=11.5,
        leading=14,
        textColor=COLOR_PRIMARY_LIGHT,
        spaceAfter=5
    )

    card_title_cyan = ParagraphStyle(
        'CardTitleCyan',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=11.5,
        leading=14,
        textColor=COLOR_CYAN,
        spaceAfter=5
    )

    card_title_emerald = ParagraphStyle(
        'CardTitleEmerald',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=11.5,
        leading=14,
        textColor=COLOR_EMERALD,
        spaceAfter=5
    )

    card_title_amber = ParagraphStyle(
        'CardTitleAmber',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=11.5,
        leading=14,
        textColor=COLOR_AMBER,
        spaceAfter=5
    )

    card_body = ParagraphStyle(
        'CardBody',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9,
        leading=13,
        textColor=COLOR_TEXT_MUTED,
    )

    code_body = ParagraphStyle(
        'CodeBody',
        parent=styles['Normal'],
        fontName='Courier',
        fontSize=8.5,
        leading=12,
        textColor=colors.HexColor("#38BDF8"),
    )

    badge_style = ParagraphStyle(
        'BadgeStyle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=8.5,
        leading=10,
        textColor=COLOR_TEXT_MAIN,
    )

    story = []

    def make_card(title_p, body_p, width=425, height=135, border_color=COLOR_CARD_BORDER, bg_color=COLOR_CARD_BG):
        content = [title_p, body_p]
        t = Table([[content]], colWidths=[width], rowHeights=[height])
        t.setStyle(TableStyle([
            ('BACKGROUND', (0,0), (-1,-1), bg_color),
            ('BOX', (0,0), (-1,-1), 1, border_color),
            ('TOPPADDING', (0,0), (-1,-1), 10),
            ('BOTTOMPADDING', (0,0), (-1,-1), 10),
            ('LEFTPADDING', (0,0), (-1,-1), 14),
            ('RIGHTPADDING', (0,0), (-1,-1), 14),
            ('VALIGN', (0,0), (-1,-1), 'TOP'),
        ]))
        return t

    # =========================================================================
    # SLIDE 1: COVER SLIDE
    # =========================================================================
    story.append(Spacer(1, 30))
    badge = Table([[Paragraph('<font color="#818CF8"><b>🚀 ENTERPRISE SUPPORT PLATFORM</b></font>', badge_style)]],
                  colWidths=[230], rowHeights=[22])
    badge.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), COLOR_PILL_BG),
        ('BOX', (0,0), (-1,-1), 1, COLOR_PRIMARY),
        ('ALIGN', (0,0), (-1,-1), 'CENTER'),
        ('VALIGN', (0,0), (-1,-1), 'MIDDLE'),
        ('LEFTPADDING', (0,0), (-1,-1), 8),
        ('RIGHTPADDING', (0,0), (-1,-1), 8),
    ]))
    story.append(badge)
    story.append(Spacer(1, 14))

    story.append(Paragraph("Modern Full-Stack Helpdesk Platform", title_style))
    story.append(Paragraph("Sub-Millisecond Query Budgets • PostgreSQL 17 • Go 1.23 • Single-Page App", subtitle_style))
    story.append(Spacer(1, 20))

    meta_table_data = [
        [
            Paragraph("<b>Author & Engineering:</b><br/><font color='#FFFFFF'>Osama Zaki</font>", card_body),
            Paragraph("<b>Core Stack:</b><br/><font color='#FFFFFF'>Go 1.23, PostgreSQL 17, Redis 7</font>", card_body),
            Paragraph("<b>Architecture:</b><br/><font color='#FFFFFF'>Transactional Outbox & Keyset SPA</font>", card_body),
            Paragraph("<b>Open Source Repository:</b><br/><font color='#818CF8'>github.com/osamazakimohammed/helpdesk</font>", card_body),
        ]
    ]
    meta_table = Table(meta_table_data, colWidths=[205, 215, 220, 240], rowHeights=[45])
    meta_table.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), COLOR_CARD_BG),
        ('BOX', (0,0), (-1,-1), 1, COLOR_CARD_BORDER),
        ('LEFTPADDING', (0,0), (-1,-1), 12),
        ('RIGHTPADDING', (0,0), (-1,-1), 12),
        ('TOPPADDING', (0,0), (-1,-1), 8),
        ('BOTTOMPADDING', (0,0), (-1,-1), 8),
        ('VALIGN', (0,0), (-1,-1), 'MIDDLE'),
    ]))
    story.append(meta_table)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 2: EXECUTIVE SUMMARY & CORE VALUE PROPOSITION
    # =========================================================================
    story.append(Paragraph("Executive Summary & Core Value Proposition", section_heading))
    story.append(Paragraph("High-performance support platform built for sub-millisecond responsiveness and simplicity.", section_subheading))

    c1 = make_card(
        Paragraph("⚡ Zero-Bloat Sub-Millisecond Speed", card_title_cyan),
        Paragraph("Eliminates heavy JS framework dependencies and slow template rendering. Strict query budget middleware enforces sub-millisecond database queries across all endpoints.", card_body),
        width=425, height=135
    )
    c2 = make_card(
        Paragraph("🔒 Uncompromising Transactional Safety", card_title_emerald),
        Paragraph("Employs the <b>Transactional Outbox Pattern</b> with atomic PostgreSQL commits. Ticket updates, audit records, and async side-effects never desynchronize or fail silently.", card_body),
        width=425, height=135
    )
    c3 = make_card(
        Paragraph("🌍 Universal Multilingual & RTL Ready", card_title_amber),
        Paragraph("Native UTF-8 support with automatic bidirectional text rendering (<code>dir='auto'</code>). Seamless right-to-left alignment for Arabic subjects, descriptions, and comments.", card_body),
        width=425, height=135
    )
    c4 = make_card(
        Paragraph("📦 Single-Binary Deployment (Self-Hostable)", card_title),
        Paragraph("The entire Single-Page Application (HTML, CSS, JS) is embedded directly into the static Go binary using <code>embed.FS</code>. One container image runs the entire system.", card_body),
        width=425, height=135
    )

    grid1 = Table([[c1, c2], [Spacer(1, 10), Spacer(1, 10)], [c3, c4]], colWidths=[435, 435])
    grid1.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid1)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 3: COMPLETE TECH STACK ARCHITECTURE
    # =========================================================================
    story.append(Paragraph("System Architecture & Technology Stack", section_heading))
    story.append(Paragraph("Engineered with compiled type safety, robust connection pooling, and minimal footprint.", section_subheading))

    s1 = make_card(
        Paragraph("⚙️ Backend Core (Go 1.23)", card_title),
        Paragraph("• <b>Chi Router:</b> Ultra-lightweight, zero-allocation HTTP router.<br/>• <b>pgx/v5 Pool:</b> High-throughput connection pooling with prepared statements.<br/>• <b>SQLC:</b> Compile-time type-safe query generation eliminating ORM overhead.<br/>• <b>JWT & Middleware:</b> Query budget enforcer, CORS, structured logging.", card_body),
        width=425, height=135
    )
    s2 = make_card(
        Paragraph("🐘 Database & Storage (PostgreSQL 17)", card_title_cyan),
        Paragraph("• <b>UUIDv7:</b> Time-ordered 128-bit UUID primary keys for natural indexing.<br/>• <b>BRIN Indexes:</b> Ultra-compact range index on high-volume audit logs.<br/>• <b>Keyset Pagination:</b> Pure (updated_at, id) cursor ordering without OFFSET.<br/>• <b>S3 Object Storage:</b> MinIO/AWS S3 compatibility with SHA-256 validation.", card_body),
        width=425, height=135
    )
    s3 = make_card(
        Paragraph("⚡ Caching & Async Workers (Redis 7)", card_title_emerald),
        Paragraph("• <b>Distributed Outbox Drainer:</b> <code>SKIP LOCKED</code> concurrent queue.<br/>• <b>Rate Limiting:</b> Token bucket protection on authentication and intake.<br/>• <b>PubSub:</b> Real-time event broadcasting and timeline synchronization.<br/>• <b>Idempotency:</b> Guaranteed deduplication on retry loops.", card_body),
        width=425, height=135
    )
    s4 = make_card(
        Paragraph("💻 Embedded Frontend (Vanilla SPA)", card_title_amber),
        Paragraph("• <b>Zero-Dependency SPA:</b> Pure vanilla JS + reactive DOM rendering.<br/>• <b>Live Hot-Reload:</b> Development mode serves disk assets instantly.<br/>• <b>Distroless Shipping:</b> Static single binary container ready for Podman/Docker.<br/>• <b>Auto RTL Alignment:</b> Native <code>dir='auto'</code> for global readability.", card_body),
        width=425, height=135
    )

    grid2 = Table([[s1, s2], [Spacer(1, 10), Spacer(1, 10)], [s3, s4]], colWidths=[435, 435])
    grid2.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid2)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 4: DEDICATED APPLICATION SURFACES
    # =========================================================================
    story.append(Paragraph("Dedicated Application Surfaces & Workflows", section_heading))
    story.append(Paragraph("Unified experiences tailored for Support Agents, Customers, and Public Users.", section_subheading))

    surf1 = make_card(
        Paragraph("🎯 Agent Workspace (<code>/app</code>)", card_title),
        Paragraph("• <b>Unified Queues:</b> All Open, Pending Response, Resolved / Closed, All Tickets.<br/>• <b>Status Transitions:</b> Real-time dropdown updates with database timestamp recording.<br/>• <b>Dual-Mode Composer:</b> Send Public Customer Replies or Save Internal Team Notes.<br/>• <b>Keyboard Shortcuts:</b> Fast triage with <i>Ctrl+Enter</i> submission.", card_body),
        width=425, height=135
    )
    surf2 = make_card(
        Paragraph("🌐 Customer Portal (<code>/portal</code>)", card_title_cyan),
        Paragraph("• <b>Self-Service Overview:</b> Instant list of all submitted tickets and live statuses.<br/>• <b>Threaded History:</b> Real-time conversation stream with agent response badges.<br/>• <b>Interactive Replies:</b> Customers reply directly into existing ticket threads.<br/>• <b>Arabic & Multilingual:</b> Clean Right-to-Left aligned text display.", card_body),
        width=425, height=135
    )
    surf3 = make_card(
        Paragraph("📝 Request Intake Form (<code>/submit</code>)", card_title_emerald),
        Paragraph("• <b>Streamlined Intake:</b> Quick ticket creation for authenticated or guest users.<br/>• <b>Priority & Categories:</b> Select issue urgency (Low, Medium, High, Urgent).<br/>• <b>Auto Identity Binding:</b> Resolves logged-in contact IDs automatically.<br/>• <b>Immediate Redirection:</b> Seamless jump to ticket view upon submission.", card_body),
        width=425, height=135
    )
    surf4 = make_card(
        Paragraph("📚 Knowledge Base & API Docs (<code>/kb</code> & <code>/api/docs</code>)", card_title_amber),
        Paragraph("• <b>Knowledge Base (<code>/kb</code>):</b> Public spaces, categories, and fast full-text search.<br/>• <b>Interactive Swagger UI (<code>/api/docs</code>):</b> Live OpenAPI 3.1 schema exploration.<br/>• <b>Universal Access:</b> Accessible for team members, clients, and developers alike.", card_body),
        width=425, height=135
    )

    grid3 = Table([[surf1, surf2], [Spacer(1, 10), Spacer(1, 10)], [surf3, surf4]], colWidths=[435, 435])
    grid3.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid3)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 5: PERFORMANCE ENGINEERING & STRICT QUERY BUDGETS
    # =========================================================================
    story.append(Paragraph("Performance Engineering & Strict Query Budgeting", section_heading))
    story.append(Paragraph("Eliminating the N+1 problem through compile-time SQL aggregation and keyset pagination.", section_subheading))

    perf1 = make_card(
        Paragraph("📊 Strict Query Budgets", card_title_cyan),
        Paragraph("• <b>Ticket Queue List: STRICTLY 1 Query</b><br/>CTE & JSON subquery aggregation bundles tags, assignee, contact, status, and organization in a single database round-trip.<br/><br/>• <b>Ticket Detail Pane: STRICTLY 2 Queries</b><br/>Query 1 fetches ticket metadata; Query 2 retrieves keyset-paginated timeline events.", card_body),
        width=425, height=135
    )
    perf2 = make_card(
        Paragraph("🚀 Keyset Pagination (No OFFSET)", card_title_emerald),
        Paragraph("• <b>Constant-Time Retrieval:</b> Uses <code>WHERE (t.updated_at, t.id) &lt; ($cursor_time, $cursor_id)</code>.<br/><br/>• <b>Zero Scan Penalty:</b> Page 1,000 loads with the exact same 0.4ms index-seek latency as Page 1.<br/><br/>• <b>Drift-Free:</b> New incoming tickets do not cause duplicate items during pagination.", card_body),
        width=425, height=135
    )
    perf3 = make_card(
        Paragraph("🛡️ Middleware Enforcement", card_title_amber),
        Paragraph("• <b>Query Budget Middleware:</b> Tracks total database queries per HTTP request.<br/><br/>• <b>Header Transparency:</b> Emits <code>X-DB-Query-Count</code> in response headers for latency & efficiency verification.<br/><br/>• <b>Circuit Breaker:</b> Flags requests exceeding the query limit.", card_body),
        width=425, height=135
    )
    perf4 = make_card(
        Paragraph("🗄️ UUIDv7 & BRIN Indexing", card_title),
        Paragraph("• <b>UUIDv7 Primary Keys:</b> Time-ordered 128-bit UUIDs prevent B-Tree fragmentation while eliminating sequential enumeration vulnerabilities.<br/><br/>• <b>BRIN Index on Audit Trail:</b> Compresses millions of immutable audit logs into tiny page-range metadata for blazing fast chronological queries.", card_body),
        width=425, height=135
    )

    grid4 = Table([[perf1, perf2], [Spacer(1, 10), Spacer(1, 10)], [perf3, perf4]], colWidths=[435, 435])
    grid4.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid4)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 6: AUTHENTICATION, ROLES & ARABIC I18N
    # =========================================================================
    story.append(Paragraph("Authentication, Roles & Internationalization", section_heading))
    story.append(Paragraph("Frictionless login, universal admin privileges, and first-class Arabic support.", section_subheading))

    auth1 = make_card(
        Paragraph("🔑 Dual Username / Email Login", card_title),
        Paragraph("• Users can authenticate using <b>either their plain username</b> (e.g. <code>admin</code>, <code>support</code>, <code>customer</code>) <b>or their full email address</b>.<br/>• Returns standard cryptographically signed JWT tokens with claims.<br/>• Automated session persistence via secure browser storage.", card_body),
        width=425, height=135
    )
    auth2 = make_card(
        Paragraph("👑 Universal Admin Access", card_title_cyan),
        Paragraph("• <b>Admin Role:</b> Direct navigation across <i>every surface</i> (Agent Workspace, Customer Portal, Submit Request, Knowledge Base, OpenAPI).<br/>• <b>Agent Role:</b> Focused ticket resolution workspace & Help Center.<br/>• <b>Customer Role:</b> Personal portal and ticket submission.", card_body),
        width=425, height=135
    )
    auth3 = make_card(
        Paragraph("🌐 Native Arabic & RTL Text Direction", card_title_emerald),
        Paragraph("• Comprehensive UTF-8 encoding across PostgreSQL, Go, and Browser.<br/>• Automatic <code>dir='auto'</code> attribute dynamically detects Arabic script and renders right-to-left alignment flawlessly.<br/>• Tested with full Arabic subjects, ticket descriptions, and agent replies.", card_body),
        width=425, height=135
    )
    auth4 = make_card(
        Paragraph("👥 Demo Accounts & Seeding", card_title_amber),
        Paragraph("• <b>Admin:</b> <code>admin</code> / <code>admin</code> (Full platform access)<br/>• <b>Support:</b> <code>support</code> / <code>support</code> (Agent workspace)<br/>• <b>Customer:</b> <code>customer</code> / <code>customer</code> (Portal & intake)<br/>• Clean seed migrations populate standard priorities and statuses.", card_body),
        width=425, height=135
    )

    grid5 = Table([[auth1, auth2], [Spacer(1, 10), Spacer(1, 10)], [auth3, auth4]], colWidths=[435, 435])
    grid5.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid5)
    story.append(PageBreak())

    # =========================================================================
    # SLIDE 7: QUICKSTART, DEPLOYMENT & SUMMARY
    # =========================================================================
    story.append(Paragraph("Quickstart, Deployment & Live Repository", section_heading))
    story.append(Paragraph("Ready to run locally with Podman/Docker or deploy as a single static binary.", section_subheading))

    q1 = make_card(
        Paragraph("🚀 1-Click Container Startup", card_title_cyan),
        Paragraph("<font color='#38BDF8'><b># Start PostgreSQL 17 & Redis 7</b><br/>podman run -d --name helpdesk-postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=helpdesk postgres:17-alpine<br/><br/># Or using Docker Compose<br/>docker compose up -d</font>", card_body),
        width=425, height=135
    )
    q2 = make_card(
        Paragraph("💻 CLI Commands & Server Launch", card_title_emerald),
        Paragraph("<font color='#38BDF8'><b># 1. Run database schema migrations</b><br/>go run cmd/server/main.go migrate<br/><br/><b># 2. Seed default roles & demo accounts</b><br/>go run cmd/server/main.go seed<br/><br/><b># 3. Start live server on :8080</b><br/>go run cmd/server/main.go serve</font>", card_body),
        width=425, height=135
    )
    q3 = make_card(
        Paragraph("📦 Cloud Deployment Options", card_title_amber),
        Paragraph("• <b>Render / Railway:</b> 1-click Git deployment from repository.<br/>• <b>Fly.io:</b> Global static distroless container deployment.<br/>• <b>Self-Hosted VPS:</b> Run on any Linux server with Docker Compose.<br/>• <b>Distroless Base:</b> Google distroless non-root security container.", card_body),
        width=425, height=135
    )
    q4 = make_card(
        Paragraph("🔗 Repository & Documentation", card_title),
        Paragraph("• <b>GitHub Repository:</b><br/><font color='#818CF8'><b>https://github.com/osamazakimohammed/helpdesk</b></font><br/><br/>• <b>Live Local Server:</b> <code>http://localhost:8080</code><br/>• <b>OpenAPI 3.1 Swagger Docs:</b> <code>http://localhost:8080/api/docs</code>", card_body),
        width=425, height=135
    )

    grid6 = Table([[q1, q2], [Spacer(1, 10), Spacer(1, 10)], [q3, q4]], colWidths=[435, 435])
    grid6.setStyle(TableStyle([('VALIGN', (0,0), (-1,-1), 'TOP')]))
    story.append(grid6)

    # Build the document with onFirstPage and onLaterPages callbacks
    doc.build(story, onFirstPage=draw_slide_background, onLaterPages=draw_slide_background)
    print(f"Presentation successfully built: {filename}")

if __name__ == "__main__":
    out = "Digitera_Helpdesk_Project_Presentation.pdf"
    if len(sys.argv) > 1:
        out = sys.argv[1]
    build_pdf(out)
